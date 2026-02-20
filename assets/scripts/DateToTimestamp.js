/**!
 * @name          Date to Timestamp
 * @description   Converts dates to Unix timestamp.
 * @icon          watch
 * @tags          date,time,calendar,unix,timestamp
 */

function main(input) {

    let parsedDate = Date.parse(input.text)

    if (isNaN(parsedDate)) {
        input.postError("Invalid Date")
    } else {
        input.text = parsedDate / 1000
    }
}
