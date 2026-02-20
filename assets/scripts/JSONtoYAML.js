/**!
 * @name          JSON to YAML
 * @description   Converts JSON to YAML.
 * @icon          metamorphose
 * @tags          markup,convert
 */

const yaml = require('@boop/js-yaml')

function main(input) {
	try {
		input.text = yaml.safeDump(JSON.parse(input.text))
	}
	catch(error) {
		input.postError("Invalid JSON")
	}
}