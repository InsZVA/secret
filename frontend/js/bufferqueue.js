/**
 * Created by InsZVA on 2017/2/3.
 */

function BufferQueue(length) {
    this._state = "fastload";
    this.length = length || 2;
    this._queue = [];
    this.onstatechange = null;
    /**
     * when chunk is ready to play it is called
     * !!!NOTE!!! the chunk is the newest in buffer
     * @type {function}
     */
    this.onchunkready = null;
}

/**
 * push a chunk to buffer
 * @param {Chunk} chunk
 */
BufferQueue.prototype.pushChunk = function(chunk) {
    this._enqueue(chunk);
    if (this._state == "fastload") {
        if (this._iscontinuous())
            this._setState("buffered");
    } else {
        if (!this._iscontinuous())
            this._setState("fastload");
        else {
            if (this.onchunkready)
                this.onchunkready(chunk);
        }
    }
};

/**
 * enqueue a chunk and remove the last if full
 * @param {Chunk} chunk
 */
BufferQueue.prototype._enqueue = function(chunk) {
    // usually length is very small, so I don't care how to implement
    var i;
    for (i = 0; i < this._queue.length; i++) {
        if (this._queue[i].id > chunk.id) break;
    }
    this._queue = this._queue.slice(0, i).concat(
        [chunk].concat(this._queue.slice(i))
    );
    if (this._queue.length > this.length) {
        this._queue = this._queue.slice(1);
    }
};

/**
 * detect the whether queue is full
 * @returns {boolean}
 */
BufferQueue.prototype._iscontinuous = function() {
    // usually length is very small, so I don't care how to implement
    for (var i = 0; i < this._queue.length - 1; i++) {
        if (this._queue[i].id + 1 != this._queue[i+1].id) return false;
    }
    return true;
};

/**
 * set the state and trigger onstatechange
 * @param state
 * @private
 */
BufferQueue.prototype._setState = function(state) {
    if (this._state == state) return;
    this._state = state;
    this.onstatechange(state);
};