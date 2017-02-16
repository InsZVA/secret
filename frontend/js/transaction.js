/**
 * Created by InsZVA on 2017/2/4.
 */

/**
 * Transaction
 * @param {string} dst - the hash of destination client
 * @param {string} msg - the description of transaction
 * @constructor
 */
function Trasaction(dst, msg) {
    this.dst = dst;
    this.msg = msg;
}