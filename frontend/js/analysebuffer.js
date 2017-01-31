/**
 * Created by InsZVA on 2017/1/29.
 */

const BUFFER_FULL_SIZE = 5000;

function AnalyseBuffer() {
    // the buffer size of this analyse buffer
    this.size = 0;
    // when the buffer is full, this event will be called
    this.onfull = null;
}

AnalyseBuffer.prototype.push = function(vs) {
    this.size += vs.nt - vs.ct;
    if (this.size >= BUFFER_FULL_SIZE && this.onfull)
        this.onfull();
};