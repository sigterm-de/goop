/**!
 * @name          Minify JSON
 * @description   Cleans and minifies JSON documents.
 * @icon          broom
 * @tags          html,minify,clean,indent
 * @bias          -0.1
 */

function main(input) {
    input.text = JSON.stringify(JSON.parse(input.text));
}
