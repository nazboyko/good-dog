// Dog vision pass: dichromatic color matrix, WebGL1 with a 2D filter
// fallback, devicePixelRatio capped at 2, see the dog-perception skill.

// Vienot deuteranopia simulation, defined for LINEAR rgb. Applied to
// srgb encoded pixels it drifts green, so the shader linearizes first.
// Each row sums to 1 so brightness never drifts, rows one and two ignore
// blue and row three ignores red, that is what dichromacy means here.
export const DOG_VISION_MATRIX: readonly number[] = [
  0.625, 0.375, 0.0,
  0.7, 0.3, 0.0,
  0.0, 0.3, 0.7,
];

// dogs see color less saturated than the simulation implies
const DESATURATION = 0.15;

const GAMMA = 2.2;

// clear to orange first, orange survives only if the draw never executed
const CLEAR_RGB = [255, 153, 0] as const;

const watchedCanvases = new WeakSet<HTMLCanvasElement>();

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
vec3 toLinear(vec3 c) { return pow(c, vec3(${GAMMA.toFixed(1)})); }
vec3 toSrgb(vec3 c) { return pow(c, vec3(1.0 / ${GAMMA.toFixed(1)})); }
void main() {
  vec4 c = texture2D(uTex, vUv);
  vec3 lin = toLinear(clamp(c.rgb, 0.0, 1.0));
  // row vector times matrix, because uniformMatrix3fv uploads column major
  vec3 dog = lin * uColorMatrix;
  float luma = dot(dog, vec3(0.2126, 0.7152, 0.0722));
  dog = mix(dog, vec3(luma), ${DESATURATION.toFixed(2)});
  gl_FragColor = vec4(toSrgb(clamp(dog, 0.0, 1.0)), c.a);
}`;

export type VisionMode = "webgl" | "2d fallback";

// Resolves only after a frame really rendered and the corner swatch strip
// passed the acceptance checks, all verified with gl.readPixels.
export async function renderDogVision(
  canvas: HTMLCanvasElement,
  imageUrl: string,
): Promise<VisionMode> {
  const image = await loadImage(imageUrl);

  const cssWidth = canvas.clientWidth;
  if (cssWidth === 0) {
    throw new Error("canvas has zero width, nothing to render into");
  }
  const dpr = Math.min(window.devicePixelRatio || 1, 2);
  canvas.width = Math.round(cssWidth * dpr);
  canvas.height = Math.round(((cssWidth * image.height) / image.width) * dpr);

  const gl = canvas.getContext("webgl");
  if (!gl) return render2dFallback(canvas, image);

  if (!watchedCanvases.has(canvas)) {
    watchedCanvases.add(canvas);
    canvas.addEventListener("webglcontextlost", (e) => e.preventDefault());
    canvas.addEventListener("webglcontextrestored", () => {
      renderDogVision(canvas, imageUrl).catch(() => {});
    });
  }

  setupPass(gl);

  gl.viewport(0, 0, gl.drawingBufferWidth, gl.drawingBufferHeight);
  gl.clearColor(CLEAR_RGB[0] / 255, CLEAR_RGB[1] / 255, CLEAR_RGB[2] / 255, 1);
  gl.clear(gl.COLOR_BUFFER_BIT);
  uploadTexture(gl, image);
  gl.drawArrays(gl.TRIANGLES, 0, 3);

  // known swatches through the same pass, small strip in the corner
  const swatch = Math.max(8, Math.round(12 * dpr));
  uploadTexture(gl, swatchPixels());
  gl.viewport(0, 0, swatch * 3, swatch);
  gl.drawArrays(gl.TRIANGLES, 0, 3);

  const glError = gl.getError();
  if (glError !== gl.NO_ERROR) {
    console.error(`dog vision draw failed, gl error ${glError}`);
  }

  const sceneCenter = readPixel(
    gl,
    Math.floor(gl.drawingBufferWidth / 2),
    Math.floor(gl.drawingBufferHeight / 2),
  );
  // the clear writes alpha 255, an untouched zero buffer means the read failed
  if (sceneCenter[3] === 0) {
    throw new Error(`frame did not render, readback came back empty, gl error ${glError}`);
  }
  if (looksLikeClearColor(sceneCenter)) {
    throw new Error(`frame did not render, center pixel is still the clear color, gl error ${glError}`);
  }

  const [red, green, blue] = [0, 1, 2].map((i) =>
    readPixel(gl, Math.floor(swatch * (i + 0.5)), Math.floor(swatch / 2)),
  );
  if (!redBecameMuddyYellow(red)) {
    throw new Error(`red swatch should be muddy yellow, got ${rgb(red)}`);
  }
  if (!greenBecamePaleYellow(green)) {
    throw new Error(`green swatch should be pale yellow, got ${rgb(green)}`);
  }
  if (!blueStayedBlue(blue)) {
    throw new Error(`blue swatch should stay blue, got ${rgb(blue)}`);
  }
  for (const p of [red, green, blue, sceneCenter]) {
    if (isGreenDominant(p)) {
      throw new Error(`dog vision must never lean green, got ${rgb(p)}`);
    }
  }
  return "webgl";
}

// Reference pipeline in plain TS, byte in byte out, mirrors the shader
// exactly so tests can pin what the GPU is supposed to do.
export function dogVisionReference(px: readonly number[]): number[] {
  const lin = px.map((v) => Math.pow(v / 255, GAMMA));
  const rows = [0, 3, 6].map((o) => DOG_VISION_MATRIX.slice(o, o + 3));
  const dog = rows.map((row) => row[0] * lin[0] + row[1] * lin[1] + row[2] * lin[2]);
  const luma = 0.2126 * dog[0] + 0.7152 * dog[1] + 0.0722 * dog[2];
  return dog.map((v) => {
    const mixed = v + (luma - v) * DESATURATION;
    return Math.round(255 * Math.pow(Math.min(Math.max(mixed, 0), 1), 1 / GAMMA));
  });
}

// Acceptance predicates shared by the live readPixels check and the tests.
export function isGreenDominant(p: ArrayLike<number>): boolean {
  return p[1] > 40 && p[1] > Math.max(p[0], p[2]) * 1.15;
}

export function redBecameMuddyYellow(p: ArrayLike<number>): boolean {
  return Math.abs(p[0] - p[1]) <= 48 && p[2] < Math.min(p[0], p[1]) * 0.6;
}

export function greenBecamePaleYellow(p: ArrayLike<number>): boolean {
  return p[0] >= 100 && p[0] + 12 >= p[1] && p[1] + 12 >= p[2] && p[0] > p[2];
}

export function blueStayedBlue(p: ArrayLike<number>): boolean {
  return p[2] > p[0] + 40 && p[2] > p[1] + 40;
}

// Tolerance covers driver rounding, not real scene pixels.
export function looksLikeClearColor(pixel: ArrayLike<number>): boolean {
  return (
    Math.abs(pixel[0] - CLEAR_RGB[0]) <= 12 &&
    Math.abs(pixel[1] - CLEAR_RGB[1]) <= 12 &&
    Math.abs(pixel[2] - CLEAR_RGB[2]) <= 12
  );
}

function setupPass(gl: WebGLRenderingContext) {
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

  gl.uniformMatrix3fv(
    gl.getUniformLocation(program, "uColorMatrix"),
    false,
    new Float32Array(DOG_VISION_MATRIX),
  );
}

// NPOT sources: clamp both axes, no mipmaps. The scene filters linear,
// the swatch strip filters nearest so cells stay flat for readback.
function uploadTexture(gl: WebGLRenderingContext, source: HTMLImageElement | Uint8Array) {
  gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, true);
  gl.bindTexture(gl.TEXTURE_2D, gl.createTexture());
  const filter = source instanceof Uint8Array ? gl.NEAREST : gl.LINEAR;
  if (source instanceof Uint8Array) {
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 3, 1, 0, gl.RGBA, gl.UNSIGNED_BYTE, source);
  } else {
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, source);
  }
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, filter);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, filter);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
}

function swatchPixels(): Uint8Array {
  return new Uint8Array([255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255]);
}

function readPixel(gl: WebGLRenderingContext, x: number, y: number): Uint8Array {
  const out = new Uint8Array(4);
  gl.readPixels(x, y, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, out);
  return out;
}

function rgb(p: ArrayLike<number>): string {
  return `rgb(${p[0]}, ${p[1]}, ${p[2]})`;
}

function render2dFallback(canvas: HTMLCanvasElement, image: HTMLImageElement): VisionMode {
  const ctx = canvas.getContext("2d")!;
  ctx.filter = "saturate(0.55) sepia(0.2)";
  ctx.drawImage(image, 0, 0, canvas.width, canvas.height);
  const center = ctx.getImageData(
    Math.floor(canvas.width / 2),
    Math.floor(canvas.height / 2),
    1,
    1,
  ).data;
  if (center[3] === 0) {
    throw new Error("2d fallback drew nothing, center pixel is still transparent");
  }
  return "2d fallback";
}

// decode() rejects broken images up front and avoids a sync decode stall
async function loadImage(url: string): Promise<HTMLImageElement> {
  const img = new Image();
  img.src = url;
  try {
    await img.decode();
  } catch {
    throw new Error(`image failed to load or decode: ${url}`);
  }
  return img;
}
