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

// what each canvas is currently showing, so a context restore repaints
// the scene that is on it now rather than the one it opened with
const latestRender = new WeakMap<HTMLCanvasElement, { imageUrl: string; showSwatches: boolean }>();

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
  { showSwatches = false }: { showSwatches?: boolean } = {},
): Promise<VisionMode> {
  // The context decides which way up the picture has to be decoded, so
  // it is asked for before the image is fetched. WebGL reads textures
  // from the bottom left, and UNPACK_FLIP_Y_WEBGL, which used to correct
  // for that, is ignored when the source is an ImageBitmap. Orientation
  // for a bitmap is fixed when it is created and nowhere else.
  const gl = canvas.getContext("webgl");
  const { image, truth } = await loadImage(imageUrl, { flipY: gl !== null });

  const cssWidth = canvas.clientWidth;
  if (cssWidth === 0) {
    throw new Error("canvas has zero width, nothing to render into");
  }
  const dpr = Math.min(window.devicePixelRatio || 1, 2);
  canvas.width = Math.round(cssWidth * dpr);
  canvas.height = Math.round(((cssWidth * image.height) / image.width) * dpr);

  if (!gl) {
    try {
      return render2dFallback(canvas, image);
    } finally {
      image.close();
    }
  }

  // the handler must repaint whatever is on this canvas now, not
  // whatever was on it the first time the listener was attached
  latestRender.set(canvas, { imageUrl, showSwatches });
  if (!watchedCanvases.has(canvas)) {
    watchedCanvases.add(canvas);
    canvas.addEventListener("webglcontextlost", (e) => e.preventDefault());
    canvas.addEventListener("webglcontextrestored", () => {
      const latest = latestRender.get(canvas);
      if (latest) renderDogVision(canvas, latest.imageUrl, { showSwatches: latest.showSwatches }).catch(() => {});
    });
  }

  setupPass(gl);

  gl.viewport(0, 0, gl.drawingBufferWidth, gl.drawingBufferHeight);
  gl.clearColor(CLEAR_RGB[0] / 255, CLEAR_RGB[1] / 255, CLEAR_RGB[2] / 255, 1);
  gl.clear(gl.COLOR_BUFFER_BIT);
  uploadTexture(gl, image);
  gl.drawArrays(gl.TRIANGLES, 0, 3);

  assertUpright(gl, truth);

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

  // The swatches are a measuring instrument, not scenery. They are read
  // off the real GPU output above and then painted over, so the frame a
  // player sees is the scene alone. The spike page keeps them on screen
  // because there the strip is the evidence.
  if (!showSwatches) {
    gl.viewport(0, 0, gl.drawingBufferWidth, gl.drawingBufferHeight);
    uploadTexture(gl, image);
    gl.drawArrays(gl.TRIANGLES, 0, 3);
    // this repaint is the frame the player actually sees, and until now
    // it was the one frame nothing looked at: a failure here left the
    // measuring strip sitting in the corner of the game
    const corner = readPixel(gl, Math.floor(swatch / 2), Math.floor(swatch / 2));
    if (redBecameMuddyYellow(corner)) {
      throw new Error(`the swatch strip survived the repaint, got ${rgb(corner)}`);
    }
  }
  image.close();
  return "webgl";
}

// The scene is on the buffer and nothing else is yet. Is it the right
// way up?
//
// Every other check here is blind to orientation, which is how an
// upside down room shipped once: the swatch strip is three texels in a
// row, so flipping it vertically does nothing, and the scene check
// reads the center pixel, which is the one pixel a vertical flip leaves
// alone. This compares the top and bottom of what was drawn against the
// top and bottom of the source, and a whole row averaged is far apart
// on any real photograph even when single pixels are not.
function assertUpright(gl: WebGLRenderingContext, truth: Edges | null) {
  if (!truth) return;
  const wantTop = dogVisionReference(truth.top);
  const wantBottom = dogVisionReference(truth.bottom);
  // gl y counts from the bottom of the buffer
  const gotTop = averageRow(gl, Math.round(gl.drawingBufferHeight * 0.94));
  const gotBottom = averageRow(gl, Math.round(gl.drawingBufferHeight * 0.06));
  const upright = apart(wantTop, gotTop) + apart(wantBottom, gotBottom);
  const flipped = apart(wantBottom, gotTop) + apart(wantTop, gotBottom);
  if (flipped < upright) {
    throw new Error(
      `the scene rendered upside down: top of frame ${rgb(gotTop)} matches the bottom of the picture`,
    );
  }
}

function averageRow(gl: WebGLRenderingContext, y: number): number[] {
  const px = new Uint8Array(4);
  const sum = [0, 0, 0];
  const n = 32;
  for (let i = 0; i < n; i++) {
    gl.readPixels(Math.round(((i + 0.5) / n) * gl.drawingBufferWidth), y, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, px);
    for (let c = 0; c < 3; c++) sum[c] += px[c];
  }
  return sum.map((v) => v / n);
}

function apart(a: readonly number[], b: readonly number[]): number {
  return Math.hypot(a[0] - b[0], a[1] - b[1], a[2] - b[2]);
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
function uploadTexture(gl: WebGLRenderingContext, source: Decoded | Uint8Array) {
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

// The 2D path is an approximation of dog vision, not the real matrix,
// and it has to be held to the same promise: no true color before the
// epilogue. Canvas filter is silently ignored where it is unsupported,
// which used to mean nothing because this only ran on the spike page.
// Now it runs behind three beats of the game, and an ignored filter
// would put the shelter on screen in full human color, which is the one
// thing the whole scene boundary exists to prevent. So it is refused
// rather than drawn wrong: a dark room is a loss, a true color one is a
// broken promise.
function render2dFallback(canvas: HTMLCanvasElement, image: Decoded): VisionMode {
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("no webgl and no 2d context, nothing can draw this scene");
  }
  if (!("filter" in ctx)) {
    throw new Error("2d fallback cannot desaturate here, refusing to draw the room in human color");
  }
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
  // the filter can also be assigned and ignored, so check the result
  if (isGreenDominant(center)) {
    throw new Error(`2d fallback left the room leaning green, got ${rgb(center)}`);
  }
  return "2d fallback";
}

// decode() rejects broken images up front and avoids a sync decode stall
type Decoded = ImageBitmap;

// Decoding must not depend on the tab being on screen. An <img> element
// decodes lazily, and img.decode() on a backgrounded document can sit
// unsettled for as long as the tab stays hidden: no error, no rejection,
// just a promise nobody hears from again and a room that never arrives.
// createImageBitmap decodes off the element and does not care. Measured
// in a hidden tab: fetch plus createImageBitmap about 15ms, img.decode()
// still unsettled after 3 seconds.
//
// flipY is not a preference, it is which consumer is asking. WebGL wants
// the rows the other way up and cannot flip a bitmap at upload time;
// the 2D fallback draws it as it comes.
async function loadImage(url: string, { flipY }: { flipY: boolean }): Promise<Loaded> {
  try {
    const res = await fetch(url);
    if (!res.ok) throw new Error(`http ${res.status}`);
    const blob = await res.blob();
    const image = await createImageBitmap(blob, {
      imageOrientation: flipY ? "flipY" : "from-image",
    });
    return { image, truth: await sourceEdges(blob) };
  } catch (e) {
    throw new Error(`image failed to load or decode: ${url}, ${e}`);
  }
}

type Loaded = { image: Decoded; truth: Edges | null };
type Edges = { top: number[]; bottom: number[] };

// The top and bottom rows of the picture as it is stored, decoded a
// second time and explicitly upright.
//
// It has to be a separate decode. Measuring the same bitmap the renderer
// is about to use tells you nothing: if that bitmap is upside down then
// so is the measurement, both sides move together and the comparison
// always agrees with itself. That mistake was made here once already.
// Decoding straight to 32x2 keeps it cheap.
async function sourceEdges(blob: Blob): Promise<Edges | null> {
  const probe = document.createElement("canvas");
  probe.width = 32;
  probe.height = 2;
  const ctx = probe.getContext("2d");
  if (!ctx) return null;
  const small = await createImageBitmap(blob, {
    imageOrientation: "from-image",
    resizeWidth: 32,
    resizeHeight: 2,
    resizeQuality: "low",
  });
  ctx.drawImage(small, 0, 0);
  small.close();
  const read = (y: number) => {
    const d = ctx.getImageData(0, y, 32, 1).data;
    const sum = [0, 0, 0];
    for (let i = 0; i < 32; i++) for (let c = 0; c < 3; c++) sum[c] += d[i * 4 + c];
    return sum.map((v) => v / 32);
  };
  return { top: read(0), bottom: read(1) };
}
