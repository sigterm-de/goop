/**!
 * @name          RGB to Hex
 * @description   Convert color in RGB to hexadecimal
 * @icon          color-wheel
 * @tags          rgb,hex,convert,color
 */

function main(state) {
  const rgb = state.text;
  const rgbArray = rgb.includes(",") ? rgb.split(",") : rgb.split(" ");

  if (rgbArray.length !== 3) return state.postError("Invalid RGB format");

  let hex = "#";

  try {
    rgbArray.forEach(c => {
      hex += parseInt(c).toString(16);
    });
  }
  catch (error) {
    return state.postError("Invalid RGB value");;
  }

  state.text = hex.toUpperCase();

}
