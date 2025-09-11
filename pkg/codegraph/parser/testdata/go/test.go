package go
// 手动刷盘
func (mb *MapBatcher) Flush() {
	mb.flush(mb.calleeMap)
}
