/**!
 * @name          Markdown Quote
 * @description   Adds > to the start of every line of your text.
 * @icon          term
 * @tags          quote,markdown
 */

function main(input) {
    input.text = input.text.split("\n").map(line => "> " + line).join("\n");
}
