/**
 * Created by InsZVA on 2017/2/7.
 */

function InputManager() {
    this._inputs = [];
}

InputManager.prototype.getAvailableNum = function () {

};

const STATISTICS_OUT_TIME = 2400;
const STATISTICS_TIME = 2000;
const FULL_PACK_NUM = 4; // 500ms per chunk && transfer by chunk
const BIT_RATE = 628*1024; // 628K bit-rate

function Input() {
    this.level = 0;
    /**
     * the input state
     * close --beginP2P--> connecting --connected--> transferring --statistics--> [ready] --need--> running/reserved
     *                      |failed|                                |out time|            |not need|
     *                        close                                  timeout               released
     * @type {string}
     */
    this.state = "close";
    // remote client id
    this.remote = null;
    // PeerConnection
    this.pc = null;
    // DataChannel
    this.dc = null;
    // WebSocket
    this.ws = null;
    // Server: masterConn
    // Client: masterConn proxy
    this.conn = null;

    // a connect operation end(success or fail)
    this.onconnectover = null;
    this.onclose = null;
    this.ontimeout = null;
    this.onrelease = null;
    this.onready = null;
    /**
     * transferred chunk num
     * @type {number}
     */
    this.transferred = 0;
}

/**
 * Dial a peer/server to get input stream
 * @param {ConnMaster} masterConn
 * @param {string} clientId
 */
Input.prototype.dial = function(masterConn, clientId) {
    if (this.state != "close") throw "Try to dial using a non-closed input.";
    if (clientId == "server") {
        this.remote = "server";
        this._masterConn = masterConn;
        this.conn = masterConn;
        this.ws = new WebSocket(LocalClient.streamAddr);
        this.ws.binaryType = 'arraybuffer';
        this.ws.onopen = function() {
            this._start(this.ws);
        }.bind(this);

        this.state = "connecting";
        this.ws.onmessage = this._transfer.bind(this);
        this.ws.onclose = function() {
            this.ws.onopen = null;
            this.ws.onclose = null;
        }.bind(this);
    } else {
        this.remote = clientId;
        this._masterConn = masterConn;
        this.conn = masterConn.newClientConn(clientId);

        this.pc = new RTCPeerConnection(LocalClient.rtcConfig);
        this.pc.onicecandidate = function (event) {
            this.conn.send({
                cmd: "icecandidate",
                candidate: event.candidate
            });
        }.bind(this);

        this.dc = this.pc.createDataChannel("inslive");
        this.dc.binaryType = 'arraybuffer';
        this.dc.onopen = function() {
            this._start(this.dc);
        }.bind(this);

        this.state = "connecting";
        this.dc.onmessage = function(e) {
            this._transfer(e);
        }.bind(this);
        this.dc.onclose = function() {
            this.dc.onopen = null;
            this.dc.onclose = null;
        }.bind(this);

        this.pc.createOffer().then(function (offer) {
            return this.pc.setLocalDescription(offer);
        }.bind(this)).then(function () {
            this.conn.onmessage = this._onmessage.bind(this);
            this.conn.send({
                cmd: "offer",
                sdp: this.pc.localDescription
            });
        }.bind(this));

    }
};

/**
 * on WebRTC signal message
 * @param {Event} e
 * @private
 */
Input.prototype._onmessage = function(e) {
    var data = e.data;
    var msg;
    try {
        msg = JSON.parse(data);
    } catch(exception) {
        return null;
    }
    if (msg.cmd && msg.cmd == "icecandidate") {
        this.pc.addIceCandidate(new RTCIceCandidate(msg.candidate))
    }
    if (msg.cmd && msg.cmd == "answer") {
        this.pc.setRemoteDescription(new RTCSessionDescription(msg.sdp));
    }
};

Input.prototype._start = function(conn) {
    this.state = "connected";
    setTimeout(this._timeout.bind(this), STATISTICS_OUT_TIME);
};

Input.prototype._transfer = function(e) {
    LocalClient.forward(e);
    var data, offset;
    if (!this.inited) this.inited = 0;

    if (this.inited < 2) {
        if (LocalClient.inited != 0) return;
        data = new Uint8Array(e.data);
        if (!this.initmsg) this.initmsg = [];
        this.initmsg[this.inited] = new InitMsg(data);
        this.inited++;
        if (this.inited == 2) {
            LocalClient.inited = 2;
            LocalClient.initmsg = [new InitMsg(this.initmsg[0].raw),
                new InitMsg(this.initmsg[1].raw)];
            LocalClient.mse = new MSE(LocalClient.videoElement,
                LocalClient.initmsg[0], LocalClient.initmsg[1]);
            console.log("init");
        }
        return
    }

    data = new Uint8Array(e.data);
    offset = bigendian.readUint32(data);
    var chunk = new Chunk(
        bigendian.readUint32(data.slice(4)),
        new Uint8Array(data.slice((offset))),
        new Uint8Array(e.data)
    );
    // if (this.remote != "server")
    //     console.log(chunk);
    var codec = new TextDecoder("utf-8").decode(data.slice(8, offset));
    if (codec == "vp9")
        LocalClient.bufferqueue[0].pushChunk(chunk);
    else
        LocalClient.bufferqueue[1].pushChunk(chunk);

    // statistics
    if (this.state == "connected") {
        this.transferred++;
        if (this.transferred >= FULL_PACK_NUM) {
            this.state = "ready";

            var incap = LocalClient.inputCap();
            if (incap > 1) {
                this.state = "released";
                if (this.onrelease)
                    this.onrelease();
            } else if (incap == 1) {
                this.state = "reserved";
                this._bind(true);
                this.dc.send(JSON.stringify({
                    reserved: true
                }));
            } else {
                this.state = "running";
                this._bind(false);
            }
        }
    }
    //TODO: slice
    //TODO: statistics
};

/**
 * evaluate the input capability and report to server
 * @param {number} value
 * @private
 */
Input.prototype._evaluate = function(value) {
    this._masterConn.send({
        type: "evaluate",
        id: this.remote,
        value: value
    });
    
};

Input.prototype._timeout = function() {
    if (this.state == "connected") {
        // Transfer capability lack
        this.state = "timeout";
        // report
        this._evaluate(0);
        if (this.ontimeout)
            this.ontimeout();
    }
};

/**
 * report to server the bind is ok
 * @param {boolean} reserved
 * @private
 */
Input.prototype._bind = function(reserved) {
    console.log(reserved);
    this._masterConn.send({
        type: "bind",
        id: this.remote,
        reserved: reserved
    });
};