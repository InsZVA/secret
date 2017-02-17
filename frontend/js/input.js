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
    this.pc = new RTCPeerConnection();
    // DataChannel
    this.dc = null;
    // WebSocket
    this.ws = null;
    // Server: masterConn
    // Client: masterConn proxy
    this.conn = null;

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
        pc.createOffer().then(function(offer) {
            return this.pc.setLocalDescription(offer);
        }.bind(this)).then(function() {
            this.conn.send({
                type: "offer",
                sdp: this.pc.localDescription
            });
            this.conn.onmessage = function(e) {
                var data = e.data;
                var msg;
                try {
                    msg = JSON.parse(data);
                } catch(exception) {
                    return null;
                }
                if (msg.type && msg.type == "answer") {
                    this.pc.setRemoteDescription(msg.sdp);
                }
            }.bind(this);
        }.bind(this));
        //TODO: create data channel
    }
};

Input.prototype._start = function(conn) {
    this.state = "connected";
    setTimeout(this._timeout, STATISTICS_OUT_TIME);
};

Input.prototype._transfer = function(e) {
    var data, offset;
    if (LocalClient.inited < 2) {
        data = new Uint8Array(e.data);
        LocalClient.initmsg[LocalClient.inited] = new InitMsg(data);
        LocalClient.inited++;
        console.log("init");
        if (LocalClient.inited == 2) {
            LocalClient.mse = new MSE(LocalClient.videoElement,
                LocalClient.initmsg[0], LocalClient.initmsg[1]);
        }
        return
    }

    data = new Uint8Array(e.data);
    offset = bigendian.readUint32(data);
    var chunk = new Chunk(
        bigendian.readUint32(data.slice(4)),
        new Uint8Array(data.slice((offset)))
    );
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
                //TODO: close conn
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
    this._masterConn.send({
        type: "bind",
        id: this.remote,
        reserved: reserved
    })
};