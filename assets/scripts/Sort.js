/**!
 * @name          Sort lines
 * @description   Sort lines alphabetically.
 * @icon          sort-characters
 * @tags          sort,alphabet
 */

function main(input) {
    let sorted = input.text.replace(/\n$/, '').split('\n')
        .sort((a, b) => a.localeCompare(b))
        .join('\n');

    if (sorted === input.text) {
        sorted = sorted.split('\n').reverse().join('\n');
    }
    input.text = sorted;
}
