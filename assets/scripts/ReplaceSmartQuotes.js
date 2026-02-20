/**!
 * @name          Replace Smart Quotes
 * @description   Replace Smart Quotes with their simpler values.
 * @icon          broom
 * @tags          smart,quotes,quotations,quotation,smart-quotes,smart-quotations
 */

function main(input) {
  input.text = input.text
    .replace(/[\u2018\u2019]/g, "'")
    .replace(/[\u201C\u201D]/g, '"')
    .replace(/“”/g, '"');
}
