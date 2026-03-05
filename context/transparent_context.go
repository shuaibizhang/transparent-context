package context

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

/*
TransparentContext 微服务链路中待透传的上下文
1、定义两类透传上下文：
1）全链路透传字段，以 Req-All-* / Resp-All-* 开头 （整个链路都透传）
2）单跳透传字段，以 Req-Once-* / Resp-Once-* 开头 (仅在一层传递， A->B)

2、此TransparentContext实现了单节点的上下文透传，包括：
1）站内Getter/Setter透传上下文字段。
2）入站时，从请求metadata中提取上下文字段，并填充到上下文中。
3）出站时，将上下文中的字段填充到请求的metadata中。
*/
type TransparentContext interface {
	/* 站内： Getter/Setter */

	// Req-All-* 全链路透传的字段信息 (请求req方向)
	GetReqAll() map[string]string
	// eg: subKey = A, 则对应透传上下文的key会填充为 Req-All-A，方便识别是全链路透传还是单跳透传
	GetReqAllByKey(subKey string) string
	SetReqAllByKey(subKey string, value string)

	// Resp-All-* 全链路透传的字段信息 (响应resp方向）
	GetRespAll() map[string]string
	GetRespAllByKey(subKey string) string
	SetRespAllByKey(subKey string, value string)

	// Req-Once-* 单跳透传的字段信息 (请求req方向)
	GetReqOnce() map[string]string
	GetReqOnceByKey(subKey string) string
	SetReqOnceByKey(subKey string, value string)

	// Resp-Once-* 单跳透传的字段信息 (响应resp方向）
	GetRespOnce() map[string]string
	GetRespOnceByKey(subKey string) string
	SetRespOnceByKey(subKey string, value string)

	/* 入站：从metadata中提取透传上下文 */
	LoadFromReqMetadata(metadata map[string]string)
	LoadFromRespMetadata(metadata map[string]string)
	/* 出站：将透传上下文中的字段填充到请求的metadata中 */
	InjectToReqMetadata() map[string]string
	InjectToRespMetadata() map[string]string
}

var _ TransparentContext = (*defaultTransparentContext)(nil)

type defaultTransparentContext struct {
	// 单个节点中请求方向继续需要向下透传的上下文字段
	reqNeedTransportMap map[string]string
	// 单个节点中响应方向无需继续向下透传的上下文字段
	reqNeedNotTransportMap map[string]string

	// 单个节点中响应方向继续需要向下透传的上下文字段
	respNeedTransportMap map[string]string
	// 单个节点中响应方向无需继续向下透传的上下文字段
	respNeedNotTransportMap map[string]string

	// 防止并发问题
	rwMutex sync.RWMutex
}

func NewTransparentContext() TransparentContext {
	return &defaultTransparentContext{}
}

func (d *defaultTransparentContext) GetReqAll() map[string]string {
	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	// Req-All-* 一定会继续向下透传，因此存放在reqNeedTransportMap中
	// Req-Once-* 单跳透传的字段信息，如果在出站前被设置，也会存放在reqNeedTransportMap中
	res := make(map[string]string)
	for k, v := range d.reqNeedTransportMap {
		if strings.HasPrefix(k, REQ_ALL_PREFIX) {
			res[k] = v
		}
	}
	return res
}

func (d *defaultTransparentContext) GetReqAllByKey(subKey string) string {
	// 使用http.CanonicalHeaderKey方法标准化key
	normalWholeKey := d.formatKeyByPrefix(REQ_ALL_PREFIX, subKey)

	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	if v, ok := d.reqNeedTransportMap[normalWholeKey]; ok {
		return v
	}

	return ""
}

func (d *defaultTransparentContext) SetReqAllByKey(subKey string, value string) {
	// 使用http.CanonicalHeaderKey方法标准化key
	normalWholeKey := d.formatKeyByPrefix(REQ_ALL_PREFIX, subKey)

	d.rwMutex.Lock()
	defer d.rwMutex.Unlock()

	if d.reqNeedTransportMap == nil {
		d.reqNeedTransportMap = make(map[string]string)
	}

	d.reqNeedTransportMap[normalWholeKey] = value
}

func (d *defaultTransparentContext) GetReqOnce() map[string]string {
	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	res := make(map[string]string)

	// 入站本服务的
	for k, v := range d.reqNeedNotTransportMap {
		if strings.HasPrefix(k, REQ_ONCE_PREFIX) {
			res[k] = v
		}
	}

	// 准备从本服务出站的
	for k, v := range d.reqNeedTransportMap {
		if strings.HasPrefix(k, REQ_ONCE_PREFIX) {
			res[k] = v
		}
	}

	return res
}

func (d *defaultTransparentContext) GetReqOnceByKey(subKey string) string {
	normalWholeKey := d.formatKeyByPrefix(REQ_ONCE_PREFIX, subKey)

	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	// 入站本服务的
	if v, ok := d.reqNeedNotTransportMap[normalWholeKey]; ok {
		return v
	}

	// 准备从本服务出站的
	if v, ok := d.reqNeedTransportMap[normalWholeKey]; ok {
		return v
	}

	return ""
}

func (d *defaultTransparentContext) SetReqOnceByKey(subKey string, value string) {
	normalWholeKey := d.formatKeyByPrefix(REQ_ONCE_PREFIX, subKey)

	d.rwMutex.Lock()
	defer d.rwMutex.Unlock()

	if d.reqNeedTransportMap == nil {
		d.reqNeedTransportMap = make(map[string]string)
	}

	// 设置待出站的key
	d.reqNeedTransportMap[normalWholeKey] = value
}
func (d *defaultTransparentContext) GetRespAll() map[string]string {
	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	res := make(map[string]string)
	for k, v := range d.respNeedTransportMap {
		if strings.HasPrefix(k, RESP_ALL_PREFIX) {
			res[k] = v
		}
	}
	return res
}

func (d *defaultTransparentContext) GetRespAllByKey(subKey string) string {
	normalWholeKey := d.formatKeyByPrefix(RESP_ALL_PREFIX, subKey)

	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	if v, ok := d.respNeedTransportMap[normalWholeKey]; ok {
		return v
	}

	return ""
}

func (d *defaultTransparentContext) SetRespAllByKey(subKey string, value string) {
	wholeKey := fmt.Sprintf("%s%s", RESP_ALL_PREFIX, subKey)
	normalWholeKey := http.CanonicalHeaderKey(wholeKey)

	d.rwMutex.Lock()
	defer d.rwMutex.Unlock()

	if d.respNeedTransportMap == nil {
		d.respNeedTransportMap = make(map[string]string)
	}

	d.respNeedTransportMap[normalWholeKey] = value
}

func (d *defaultTransparentContext) GetRespOnce() map[string]string {
	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	res := make(map[string]string)

	// 入站本服务的（无需透传）
	for k, v := range d.respNeedNotTransportMap {
		if strings.HasPrefix(k, RESP_ONCE_PREFIX) {
			res[k] = v
		}
	}

	// 准备从本服务出站的（当前跳设置，下跳不再透传）
	for k, v := range d.respNeedTransportMap {
		if strings.HasPrefix(k, RESP_ONCE_PREFIX) {
			res[k] = v
		}
	}

	return res
}

func (d *defaultTransparentContext) GetRespOnceByKey(subKey string) string {
	normalWholeKey := d.formatKeyByPrefix(RESP_ONCE_PREFIX, subKey)

	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	// 查找入站的单次响应数据
	if v, ok := d.respNeedNotTransportMap[normalWholeKey]; ok {
		return v
	}

	// 查找出站方向上的单次响应数据
	if v, ok := d.respNeedTransportMap[normalWholeKey]; ok {
		return v
	}

	return ""
}

func (d *defaultTransparentContext) SetRespOnceByKey(subKey string, value string) {
	normalWholeKey := d.formatKeyByPrefix(RESP_ONCE_PREFIX, subKey)

	d.rwMutex.Lock()
	defer d.rwMutex.Unlock()

	if d.respNeedTransportMap == nil {
		d.respNeedTransportMap = make(map[string]string)
	}

	d.respNeedTransportMap[normalWholeKey] = value
}

// 入站
func (d *defaultTransparentContext) LoadFromReqMetadata(reqMetadata map[string]string) {
	if reqMetadata == nil {
		reqMetadata = make(map[string]string)
	}

	// 标准化
	for k, v := range reqMetadata {
		normalWholeKey := http.CanonicalHeaderKey(strings.TrimSpace(k))
		reqMetadata[normalWholeKey] = v
	}

	// 入站时，根据reqMetadata中的key的类型，填充到相应map中（确认是否需要继续透传）
	for k, v := range reqMetadata {
		if strings.HasPrefix(k, REQ_ALL_PREFIX) {
			if d.reqNeedTransportMap == nil {
				d.reqNeedTransportMap = make(map[string]string)
			}
			d.reqNeedTransportMap[k] = v
		}

		if strings.HasPrefix(k, REQ_ONCE_PREFIX) {
			if d.reqNeedNotTransportMap == nil {
				d.reqNeedNotTransportMap = make(map[string]string)
			}
			d.reqNeedNotTransportMap[k] = v
		}
	}
}

// 出站
func (d *defaultTransparentContext) InjectToReqMetadata() map[string]string {
	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	metadata := make(map[string]string)

	// 需要透传的所有key都继续传下去
	for k, v := range d.reqNeedTransportMap {
		metadata[k] = v
	}

	return metadata
}

// 入站
func (d *defaultTransparentContext) LoadFromRespMetadata(respMetadata map[string]string) {
	if respMetadata == nil {
		respMetadata = make(map[string]string)
	}

	// 标准化
	for k, v := range respMetadata {
		normalWholeKey := http.CanonicalHeaderKey(strings.TrimSpace(k))
		respMetadata[normalWholeKey] = v
	}

	// 入站时，根据respMetadata中的key的类型，填充到相应map中（确认是否需要继续透传）
	for k, v := range respMetadata {
		if strings.HasPrefix(k, RESP_ALL_PREFIX) {
			if d.respNeedTransportMap == nil {
				d.respNeedTransportMap = make(map[string]string)
			}
			d.respNeedTransportMap[k] = v
		}

		if strings.HasPrefix(k, RESP_ONCE_PREFIX) {
			if d.respNeedNotTransportMap == nil {
				d.respNeedNotTransportMap = make(map[string]string)
			}
			d.respNeedNotTransportMap[k] = v
		}
	}
}

// 出站
func (d *defaultTransparentContext) InjectToRespMetadata() map[string]string {
	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()

	metadata := make(map[string]string)

	// 需要透传的所有key都继续传下去
	for k, v := range d.respNeedTransportMap {
		metadata[k] = v
	}

	return metadata
}

func (d *defaultTransparentContext) formatKeyByPrefix(prefix, key string) string {
	return http.CanonicalHeaderKey(fmt.Sprintf("%s%s", prefix, key))
}
