/**
 * Created by InsZVA on 2017/1/28.
 */
/*
    All nodes (include tracker) hold a video buffer that cached a period of time
    (eg. 5s) of video to survive network area. In the whole network, the buffer
    is similar. And all nodes are playing the video created 5s ago.

    Expect the beginning 5s with the first few nodes in this network, which
    need continue pending and buffering these video, all nodes only download
    these buffer by a fast speed (as we all know, download speed is usually much
    faster than video play speed). So most nodes will start quickly rather than
    5s.

    About the fast load: when the buffer lost the source, it can be in a lost state,
    to restore the lost buffer, the above program will try to get buffers from many
    source, for example:
    | T - 8s | T - 7s | T - 6s | are now in buffer, above should request segments start
    from | T - 5s| from other sources. If there'are 4 available candidate source,
    each source will send its buffered segments after | T - 5s | (included) in a random
    order. The above program will put them in main buffer to make main buffer full as
    faster as possible. This is called "Fast Load".
 */
const CACHED_DURATION = 5 * 1000;

/**
 * VideoBuffer represents a buffer defined in design doc
 * @constructor
 */
function VideoBuffer() {
    // the buffer of video slices
    this.buffer = [];
    // the cached duration
    this.bufferedDuration = CACHED_DURATION;
    /**
     * the state of video buffer, which is buffering at begin,
     * but to buffered once buffer is full. when the buffer is
     * removed to empty, the state will be die. when it is die,
     * pushing video data in it can switch state to buffering.
     * @type {string}
     */
    this.state = "buffering";
    // when state changed, the event will be called
    // function(state)
    this.onstatechanged = null;
    // fast load buffer
    this.flbuffer = new PriorityQueue(function(a, b) {
        return b.ct - a.ct;
    });
    // when data ready (buffer is full or not), the event will be called
    // ready means the data.createtime is out
    this.ondataready = null;
    // when in fastload period, this variable save the timestamp,
    // it can calculate newest create timestamp according to this
    this.fastloadct = 0;

    setInterval(this.dataReady.bind(this), 50);
}

/**
 * VideoSlice describe a video slice and its created ct
 * @param ct {number} - create timestamp
 * @param nt {number} - next slice timestamp
 * @param data {Uint8Array}
 * @constructor
 */
function VideoSlice(ct, nt, data) {
    this.ct = ct;
    this.nt = nt;
    this.data = data;
}

/**
 * setState function
 * set buffer state and call onstatechanged handler
 * @param newState
 */
VideoBuffer.prototype.setState = function(newState) {
    if (this.state == newState) return;
    this.state = newState;
    if (this.onstatechanged)
        this.onstatechanged(this.state);
};

/**
 * fastload function
 * this function will be called by above program.
 * then the buffer go in fastload period. the newest timer
 * start.
 */
VideoBuffer.prototype.fastload = function() {
    this.fastloadct = (this.buffer.length > 0 ?
            this.buffer[this.buffer.length - 1].nt : Infinity)
            - (new Date().getTime());
    this.setState("buffering");
};

/**
 * update function
 * remove too old buffer like below:
 * | T - 7s | T - 6s | T - 5s | T - 4s | T - 3s| T - 2s | ...
 * | T - 7s | is too old and to be removed. And | T - 1s |,
 * | T - 0s | maybe in transferring.
 * NOTE: returns array
 * @return {Array<VideoSlice>} video data
 */
VideoBuffer.prototype.update = function() {
    var newest = this.getNewestCTimestamp();
    for (var i = 0; i < this.buffer.length; i++) {
        if (this.buffer[i].ct >= newest - this.bufferedDuration) break;
    }
    var ret = this.buffer.slice(0, i);
    this.buffer = this.buffer.slice(i);
    if (this.buffer.length == 0) {
        //this.setState("die");
    }
    return ret;
};

VideoBuffer.prototype.getBufferedDuration = function() {
    if (this.buffer.length == 0) return 0;
    return this.buffer[this.buffer.length - 1].nt - this.buffer[0].ct;
};

VideoBuffer.prototype.getNewestCTimestamp = function() {
    if (this.state == "buffering")
        return (new Date().getTime()) - this.fastloadct;

    if (this.buffer.length == 0) return 0;
    return this.buffer[this.buffer.length - 1].nt;
};

/**
 * pushVideoSlice function
 * push a video slice to this buffer, but it may be unordered using the
 * fast load buffer tech: when source lost,
 * @param vs {VideoSlice}
 */
VideoBuffer.prototype.pushVideoSlice = function(vs) {
    if (this.state == "buffering") {
        // FastLoad
        console.log("FastLoad:", vs);
        this.flbuffer.enq(vs);

        // when fast loading get a continuous buffer
        // push them to buffer
        // FastLoad source must send the last request buffer slice first
        // others random
        while (!this.flbuffer.isEmpty() && (this.buffer.length == 0 ||
            this.flbuffer.peek().ct == this.buffer[this.buffer.length - 1].nt)) {
            this.buffer.push(this.flbuffer.deq());
        }

        // flbuffer duration is zero
        if (this.getBufferedDuration() >= CACHED_DURATION) {
            while (!this.flbuffer.isEmpty()) {
                this.buffer.push(this.flbuffer.deq());
            }
            this.setState("buffered");
        }
    } else if (this.state == "buffered") {
        console.log("buffered:", vs);
        this.buffer.push(vs);
    } else {
        this.buffer.push(vs);
        this.fastload();
    }
    this.dataReady();
};

VideoBuffer.prototype.dataReady = function() {
    var vsArray = this.update();
    for (var i = 0; i < vsArray.length; i++) {
        if (this.ondataready)
            this.ondataready(vsArray[i].data);
    }
};