/**
 * Created by InsZVA on 2017/2/4.
 */

function Client(config) {
    if (!config) config = {};
    this.state = "close";

    this.onready = null;
    this.inputs = null;

    // Config
    this.masterConn = new ConnMaster((config.master || 'ws://127.0.0.1:8888/master'));
    this.transmode = config.transmode || "chunk";
    this.rtcConfig = config.rtcConfig || {
            iceServers: [{
                urls: [
                    //"stun:stun.example.com"
                ]
            }]
        };
    this.bufferLength = config.bufferLength || 4;
    this.videoElement = config.videoElement;
    if (!this.videoElement)
        throw "A video element is necessary";

    this.masterConn.onmessage = this.onmastermessage;
    this.masterConn.onopen = function() {
        this.state = "ready";
        this.masterConn.send({type: "getid"});
        if (this.onready) this.onready();
    }.bind(this);

    this.bufferqueue = [new BufferQueue(this.bufferLength),
        new BufferQueue(this.bufferLength)];
    this.mse = undefined;
    this.inited = 0;
    this.initmsg = [];

    if (this.transmode == "chunk") {
        this.bufferqueue[0].onstatechange = function(state) {
            console.log(state);
        };
        this.bufferqueue[1].onstatechange = function(state) {
            console.log(state);
        };
        this.bufferqueue[0].onchunkready = function(chunk) {
            this.mse.syncChunk(0, chunk);
        }.bind(this);
        this.bufferqueue[1].onchunkready = function(chunk) {
            this.mse.syncChunk(1, chunk);
        }.bind(this);
    } else {
        //TODO: slice
    }

    LocalClient = window.LocalClient = this;
}

/**
 * get a input that can be used to start a transaction
 * it may be from a current exist input or new input
 * @returns {Array<Input>}
 */
Client.prototype.getReusedInputs = function(n) {
    var i;
    var ret = [];
    if (this.inputs == null) {
        this.inputs = [];
        for (i = 0; i < n; i++)
        {
            this.inputs.push(new Input());
            ret.push(this.inputs[i]);
        }
        return ret;
    }

    for (i = 0; i < this.inputs.length; i++) {
        if (this.inputs[i].state == "close")
        {
            ret.push(this.inputs[i]);
            if (ret.length == n)
                return ret;
        }
    }

    for (i = ret.length; i < n; i++) {
        ret.push(new Input());
        this.inputs.push(ret[i]);
    }
    return ret;
};

Client.prototype.onmastermessage = function(e) {
    var data;
    try {
        data = JSON.parse(e.data);
    } catch (exception) {
        return this.onerrormessage(e);
    }
    switch (data.type) {
        case "id":
            this.id = data.id;
            break;
        case "peek":
            var peeked = data.peek;
            var inputs = LocalClient.getReusedInputs(peeked.length);
            for (var i = 0; i < peeked.length; i++) {
                inputs[i].dial(LocalClient.masterConn, peeked[i]);
            }

            //TODO
    }
};

Client.prototype.onerrormessage = function(e) {
    throw e;
};

Client.prototype.connect = function() {
    if (this.state == "close")
        this.masterConn.connect();
    else
        throw "double connect error"
};

Client.prototype.find = function() {
    if (this.state == "transaction") throw "Client is on transaction.";
    if (this.inputCap() < 2) {
        if (this.runningInput() < 0 ||
            this.inputs[this.runningInput()].remote != "server")
        {
            this.masterConn.send({type: "find"});
            this.state = "transaction";
        }
    }
};

Client.prototype.inputCap = function() {
    if (!this.inputs) return 0;
    var ret = 0;
    for (var i = 0; i < this.inputs.length; i++) {
        if (this.inputs[i].state == "running" || this.inputs[i].
        state == "reserved")
        {
            if (this.inputs[i].remote == "server")
                ret++;
            ret++;
        }
    }
    return ret;
};

/**
 * find and return the running input index
 * @return {number}
 */
Client.prototype.runningInput = function() {
    if (!this.inputs) return -1;
    for (var i = 0; i < this.inputs.length; i++) {
        if (this.inputs[i].state == "running")
            return i;
    }
    return -1;
};

var LocalClient;
//var LocalClient = window.LocalClient = new Client();