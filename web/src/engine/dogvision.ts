// Dog vision pass: dichromatic color matrix, WebGL1 with a 2D filter
// fallback, devicePixelRatio capped at 2, see the dog-perception skill.

// Deuteranope style collapse: red and green fold toward yellow, blue stays.
// Each row sums to 1 so brightness never drifts, rows one and two ignore
// blue and row three ignores red, that is what dichromacy means here.
export const DOG_VISION_MATRIX: readonly number[] = [
  0.625, 0.375, 0.0,
  0.7, 0.3, 0.0,
  0.0, 0.3, 0.7,
];

const VERT = `
attribute vec2 aPos;
varying vec2 vUv;
void main() {
  vUv = aPos * 0.5 + 0.5;
  gl_Position = vec4(aPos, 0.0, 1.0);
}`;

const FRAG = `
precision mediump float;
varying vec2 vUv;
uniform sampler2D uTex;
uniform mat3 uColorMatrix;
void main() {
  vec4 c = texture2D(uTex, vUv);
  // row vector times matrix, because uniformMatrix3fv uploads column major
  gl_FragColor = vec4(c.rgb * uColorMatrix, c.a);
}`;

export type VisionMode = "webgl" | "2d fallback";

export async function renderDogVision(
  canvas: HTMLCanvasElement,
  imageUrl: string,
): Promise<VisionMode> {
  const image = await loadImage(imageUrl);
  const dpr = Math.min(window.devicePixelRatio || 1, 2);
  canvas.width = Math.round(canvas.clientWidth * dpr);
  canvas.height = Math.round((canvas.clientWidth * image.height) / image.width * dpr);

  const gl = canvas.getContext("webgl");
  if (!gl) return render2dFallback(canvas, image);

  if (!watchedCanvases.has(canvas)) {
    watchedCanvases.add(canvas);
    canvas.addEventListener("webglcontextlost", (e) => e.preventDefault());
    canvas.addEventListener("webglcontextrestored", () => {
      renderDogVision(canvas, imageUrl).catch(() => {});
    });
  }

  drawWithMatrix(gl, image);
  return "webgl";
}

const watchedCanvases = new WeakSet<HTMLCanvasElement>();

function drawWithMatrix(gl: WebGLRenderingContext, image: HTMLImageElement) {
  const program = gl.createProgram()!;
  for (const [type, src] of [
    [gl.VERTEX_SHADER, VERT],
    [gl.FRAGMENT_SHADER, FRAG],
  ] as const) {
    const shader = gl.createShader(type)!;
    gl.shaderSource(shader, src);
    gl.compileShader(shader);
    if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
      throw new Error(`shader compile failed: ${gl.getShaderInfoLog(shader)}`);
    }
    gl.attachShader(program, shader);
  }
  gl.linkProgram(program);
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    throw new Error(`program link failed: ${gl.getProgramInfoLog(program)}`);
  }
  gl.useProgram(program);

  gl.bindBuffer(gl.ARRAY_BUFFER, gl.createBuffer());
  gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), gl.STATIC_DRAW);
  const aPos = gl.getAttribLocation(program, "aPos");
  gl.enableVertexAttribArray(aPos);
  gl.vertexAttribPointer(aPos, 2, gl.FLOAT, false, 0, 0);

  gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, true);
  gl.bindTexture(gl.TEXTURE_2D, gl.createTexture());
  gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, image);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);

  gl.uniformMatrix3fv(
    gl.getUniformLocation(program, "uColorMatrix"),
    false,
    new Float32Array(DOG_VISION_MATRIX),
  );
  gl.viewport(0, 0, gl.drawingBufferWidth, gl.drawingBufferHeight);
  gl.drawArrays(gl.TRIANGLES, 0, 3);
}

function render2dFallback(canvas: HTMLCanvasElement, image: HTMLImageElement): VisionMode {
  const ctx = canvas.getContext("2d")!;
  ctx.filter = "saturate(0.55) sepia(0.2)";
  ctx.drawImage(image, 0, 0, canvas.width, canvas.height);
  return "2d fallback";
}

function loadImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error(`image failed to load: ${url}`));
    img.src = url;
  });
}
