import { AvatarViewer } from '/static/js/profile/profile.js';
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js';
import { OBJLoader } from 'three/addons/loaders/OBJLoader.js';
import { OBJExporter } from 'three/addons/exporters/OBJExporter.js';

const THREE = window.THREE;

const LIMB_ORDER = ['head', 'torso', 'larm', 'rarm', 'lleg', 'rleg'];
const HIT_RADIUS = 28;

const raycaster = new THREE.Raycaster();
const dragPlane = new THREE.Plane();

function defaultTransform() {
    return {
        position: { x: 0, y: 0, z: 0 },
        rotation: { x: 0, y: 0, z: 0 },
        scale: { x: 1, y: 1, z: 1 }
    };
}

const SNAPS = {
    'head-top': { limb: 'head', surface: 'top' },
    'head-front': { limb: 'head', surface: 'front' },
    'torso-top': { limb: 'torso', surface: 'top' },
    'torso-front': { limb: 'torso', surface: 'front' },
    'larm-top': { limb: 'larm', surface: 'top' },
    'larm-hand': { limb: 'larm', surface: 'hand' },
    'rarm-top': { limb: 'rarm', surface: 'top' },
    'rarm-hand': { limb: 'rarm', surface: 'hand' },
    'lleg-top': { limb: 'lleg', surface: 'top' },
    'lleg-foot': { limb: 'lleg', surface: 'foot' },
    'rleg-top': { limb: 'rleg', surface: 'top' },
    'rleg-foot': { limb: 'rleg', surface: 'foot' }
};

function defaultSnap() {
    return 'none';
}

function getCurrentUserId() {
    const img = document.querySelector('.pfpimg');
    if (img) {
        const m = img.src.match(/\/headshot\/(\d+)\.png/);
        if (m) return parseInt(m[1], 10);
    }
    return 0;
}

function makeDefaultMaterial() {
    return new THREE.MeshStandardMaterial({ color: 0xffffff, roughness: 0.8, metalness: 0.05 });
}

function intersectRayPlane(ray, plane, target) {
    const denom = plane.normal.dot(ray.direction);
    if (Math.abs(denom) < 1e-8) return null;
    const t = -(plane.normal.dot(ray.origin) + plane.constant) / denom;
    if (t < 0) return null;
    target.copy(ray.direction).multiplyScalar(t).add(ray.origin);
    return target;
}

function loadModelFile(file) {
    const ext = (file.name.split('.').pop() || '').toLowerCase();
    const url = URL.createObjectURL(file);
    let promise;
    if (ext === 'glb') {
        promise = new GLTFLoader().loadAsync(url).then(gltf => gltf.scene);
    } else {
        promise = new OBJLoader().loadAsync(url);
    }
    return promise.finally(() => URL.revokeObjectURL(url));
}

async function extractModelTexture(model) {
    let texture = null;
    model.traverse(child => {
        if (texture || !child.isMesh) return;
        const mats = Array.isArray(child.material) ? child.material : [child.material];
        for (const m of mats) {
            if (m && m.map) {
                texture = m.map;
                return;
            }
        }
    });
    if (!texture || !texture.image) return null;
    const img = texture.image;
    const isDrawable = (typeof ImageBitmap !== 'undefined' && img instanceof ImageBitmap) ||
        (typeof HTMLImageElement !== 'undefined' && img instanceof HTMLImageElement) ||
        (typeof HTMLCanvasElement !== 'undefined' && img instanceof HTMLCanvasElement);
    if (!isDrawable) return null;
    const width = img.naturalWidth || img.width;
    const height = img.naturalHeight || img.height;
    if (!width || !height) return null;
    const canvas = document.createElement('canvas');
    canvas.width = width;
    canvas.height = height;
    canvas.getContext('2d').drawImage(img, 0, 0);
    return new Promise(resolve => canvas.toBlob(blob => resolve(blob), 'image/png'));
}

class ModelPreview {
    constructor(options = {}) {
        this.viewer = null;
        this.pivot = null;
        this.limbAnchors = {};
        this.limbParts = {};
        this.container = null;
        this.dragState = null;
        this.transform = null;
        this.snap = null;
        this.modelFile = null;
        this.textureFile = null;
        this.cat = null;
        this.showError = options.showError || null;
        this.refreshTimer = null;
        this.refreshSeq = 0;

        this.onViewPointerDown = e => this.handlePointerDown(e);
        this.onWindowPointerMove = e => this.handlePointerMove(e);
        this.onWindowPointerUp = e => this.handlePointerUp(e);
    }

    getTransform() {
        if (!this.transform) {
            this.transform = defaultTransform();
        }
        const t = this.transform;
        if (typeof t.scale !== 'object' || t.scale === null) {
            const s = t.scale || 1;
            t.scale = { x: s, y: s, z: s };
        }
        return t;
    }

    setModel(file) {
        this.modelFile = file || null;
    }

    setTexture(file) {
        this.textureFile = file || null;
    }

    setTransform(transform) {
        if (transform === null || transform === undefined) {
            this.transform = null;
            return;
        }
        this.transform = transform;
        if (typeof transform.scale !== 'object' || transform.scale === null) {
            const s = transform.scale || 1;
            transform.scale = { x: s, y: s, z: s };
        }
    }

    setCategory(cat) {
        this.cat = cat;
    }

    get modal() {
        return document.getElementById('crt3dModal');
    }

    get viewEl() {
        return document.getElementById('crt3dView');
    }

    get canvas() {
        return this.viewer ? this.viewer.renderer.domElement : null;
    }

    get snapValue() {
        return this.snap || defaultSnap();
    }

    get weld() {
        const info = SNAPS[this.snapValue];
        return info ? info.limb : 'head';
    }

    setSnapValue(value) {
        const sel = document.getElementById('crt3dSnap');
        if (sel) sel.value = value;
    }

    syncInputs() {
        const t = this.getTransform();
        const set = (id, v) => {
            const el = document.getElementById(id);
            if (el) el.value = Math.round(v * 100) / 100;
        };
        set('crt3dPosX', t.position.x);
        set('crt3dPosY', t.position.y);
        set('crt3dPosZ', t.position.z);
        set('crt3dRotX', t.rotation.x);
        set('crt3dRotY', t.rotation.y);
        set('crt3dRotZ', t.rotation.z);
        set('crt3dScaleX', t.scale.x);
        set('crt3dScaleY', t.scale.y);
        set('crt3dScaleZ', t.scale.z);
    }

    readInputs() {
        const t = this.getTransform();
        const get = id => {
            const el = document.getElementById(id);
            const v = el ? parseFloat(el.value) : NaN;
            return isNaN(v) ? 0 : v;
        };
        t.position.x = get('crt3dPosX');
        t.position.y = get('crt3dPosY');
        t.position.z = get('crt3dPosZ');
        t.rotation.x = get('crt3dRotX');
        t.rotation.y = get('crt3dRotY');
        t.rotation.z = get('crt3dRotZ');
        t.scale.x = Math.max(0.05, get('crt3dScaleX'));
        t.scale.y = Math.max(0.05, get('crt3dScaleY'));
        t.scale.z = Math.max(0.05, get('crt3dScaleZ'));
        return t;
    }

    getAttachmentPosition(snap) {
        if (snap === 'none') return { x: 0, y: 0, z: 0 };
        const info = SNAPS[snap] || SNAPS[defaultSnap()];
        const limb = this.limbAnchors[info.limb] || this.limbAnchors.head;
        if (!limb) return { x: 0, y: 0, z: 0 };
        const surface = info.surface;
        if (surface === 'top') {
            return { x: limb.center.x, y: limb.max.y, z: limb.center.z };
        }
        if (surface === 'hand' || surface === 'foot') {
            const end = limb.end || new THREE.Vector3(limb.center.x, limb.min.y, limb.center.z);
            return { x: end.x, y: end.y, z: end.z };
        }
        return { x: limb.center.x, y: limb.center.y, z: limb.center.z + limb.size.z * 0.35 };
    }

    applyTransform() {
        if (!this.pivot) return;
        const t = this.getTransform();
        const base = this.getAttachmentPosition(this.snapValue);
        const baseScale = this.pivot.userData.baseScale || 1;
        const lift = this.snapValue === 'none' ? 0 : (this.pivot.userData.lift || 0);

        this.pivot.scale.set(
            baseScale * t.scale.x,
            baseScale * t.scale.y,
            baseScale * t.scale.z
        );
        this.pivot.rotation.set(
            THREE.MathUtils.degToRad(t.rotation.x),
            THREE.MathUtils.degToRad(t.rotation.y),
            THREE.MathUtils.degToRad(t.rotation.z)
        );
        this.pivot.position.set(
            base.x + t.position.x,
            base.y + lift * t.scale.y + t.position.y,
            base.z + t.position.z
        );
    }

    scheduleRefresh() {
        if (this.refreshTimer) clearTimeout(this.refreshTimer);
        this.refreshTimer = setTimeout(() => {
            this.refreshTimer = null;
            this.refreshPreview();
        }, 350);
    }

    showPreviewSpinner(show) {
        const wrap = document.querySelector('#crtShopPreview .avtitmimg');
        if (!wrap) return;
        if (show) {
            if (wrap.querySelector('.musld')) return;
            const spinner = document.createElement('div');
            spinner.className = 'musld';
            spinner.style.position = 'absolute';
            spinner.style.inset = '0';
            spinner.style.zIndex = '10';
            spinner.style.pointerEvents = 'none';
            spinner.innerHTML = `
                <svg class="dmspn" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
                    <rect class="sq sq-1" x="43" y="7" width="14" height="14"/>
                    <rect class="sq sq-2" x="25" y="25" width="14" height="14"/>
                    <rect class="sq sq-3" x="7" y="43" width="14" height="14"/>
                    <rect class="sq sq-4" x="25" y="61" width="14" height="14"/>
                    <rect class="sq sq-5" x="43" y="79" width="14" height="14"/>
                    <rect class="sq sq-6" x="61" y="61" width="14" height="14"/>
                    <rect class="sq sq-7" x="79" y="43" width="14" height="14"/>
                    <rect class="sq sq-8" x="61" y="25" width="14" height="14"/>
                </svg>
            `;
            wrap.appendChild(spinner);
        } else {
            const spinner = wrap.querySelector('.musld');
            if (spinner) spinner.remove();
        }
    }

    async refreshPreview() {
        const img = document.getElementById('crtShopPreviewImg');
        if (!img) return;

        let cat = this.cat;
        if (!cat) {
            const catTab = document.querySelector('.crtshoptabs .tabbtn.active');
            if (catTab && catTab.dataset.cat) {
                cat = catTab.dataset.cat;
                this.cat = cat;
            }
        }
        if (!cat) return;

        const isModel = cat === 'hat' || cat === 'face' || cat === 'tool';
        if (!isModel && cat !== 'shirt' && cat !== 'tshirt' && cat !== 'pants') return;
        if (isModel && !this.modelFile) return;
        if (!isModel && !this.textureFile) return;

        const seq = ++this.refreshSeq;
        this.showPreviewSpinner(true);

        let file = null;
        let textureFile = null;
        if (isModel) {
            try {
                const model = await loadModelFile(this.modelFile);
                const t = this.getTransform();
                const wrapper = new THREE.Group();
                wrapper.add(model);
                wrapper.position.set(t.position.x, t.position.y, t.position.z);
                wrapper.rotation.set(
                    THREE.MathUtils.degToRad(t.rotation.x),
                    THREE.MathUtils.degToRad(t.rotation.y),
                    THREE.MathUtils.degToRad(t.rotation.z)
                );
                wrapper.scale.set(t.scale.x, t.scale.y, t.scale.z);
                wrapper.updateMatrixWorld(true);
                file = new File([new OBJExporter().parse(wrapper)], 'model.obj', { type: 'text/plain' });
                textureFile = this.textureFile || (await extractModelTexture(model));
            } catch {
                this.showPreviewSpinner(false);
                return;
            }
        } else {
            file = this.textureFile;
        }

        const formData = new FormData();
        formData.append('category', cat);
        formData.append('file', file);
        if (textureFile) formData.append('texture', textureFile);

        img.classList.remove('ld');

        try {
            const res = await fetch('/api/v1/create/preview', { method: 'POST', body: formData });
            if (!res.ok) {
                img.classList.add('ld');
                return;
            }
            const blob = await res.blob();
            if (seq !== this.refreshSeq) return;
            const url = URL.createObjectURL(blob);
            img.onload = () => {
                img.classList.add('ld');
                URL.revokeObjectURL(url);
            };
            img.src = url;
        } catch {
            img.classList.add('ld');
            return;
        } finally {
            if (seq === this.refreshSeq) this.showPreviewSpinner(false);
        }
    }

    computeLimbAnchors() {
        this.limbAnchors = {};
        this.limbParts = {};
        const group = this.viewer.avatarGroup;
        group.updateWorldMatrix(true, true);
        const inv = new THREE.Matrix4().copy(group.matrixWorld).invert();
        const parts = group.children.filter(child => child !== this.pivot);

        if (parts.length >= 6) {
            this.buildAnchors(this.computeBoxes(parts, inv));
        } else {
            this.buildFallbackAnchors(parts, inv);
        }

        this.computeEndpoints();
    }

    computeBoxes(parts, inv) {
        const boxes = {};
        parts.slice(0, 6).forEach((part, i) => {
            const box = new THREE.Box3().setFromObject(part).applyMatrix4(inv);
            if (box.isEmpty()) return;
            const key = LIMB_ORDER[i];
            part.userData.limbKey = key;
            this.limbParts[key] = part;
            boxes[key] = box;
        });
        return boxes;
    }

    buildAnchors(boxes) {
        for (const [key, box] of Object.entries(boxes)) {
            this.limbAnchors[key] = {
                center: box.getCenter(new THREE.Vector3()),
                size: box.getSize(new THREE.Vector3()),
                min: box.min.clone(),
                max: box.max.clone()
            };
        }
    }

    buildFallbackAnchors(parts, inv) {
        const box = new THREE.Box3();
        parts.forEach(part => box.expandByObject(part));
        if (box.isEmpty()) return;
        box.applyMatrix4(inv);
        const c = box.getCenter(new THREE.Vector3());
        const s = box.getSize(new THREE.Vector3());

        const anchor = (center, size, min, max) => ({
            center,
            size: size.clone(),
            min: min.clone(),
            max: max.clone()
        });

        this.limbAnchors = {
            head: {
                center: new THREE.Vector3(c.x, box.max.y - s.y * 0.1, c.z),
                size: new THREE.Vector3(s.x * 0.4, s.y * 0.18, s.z * 0.4),
                min: new THREE.Vector3(c.x - s.x * 0.2, box.max.y - s.y * 0.18, c.z - s.z * 0.2),
                max: new THREE.Vector3(c.x + s.x * 0.2, box.max.y, c.z + s.z * 0.2)
            },
            torso: anchor(new THREE.Vector3(c.x, c.y + s.y * 0.05, c.z), s, box.min, box.max),
            larm: anchor(new THREE.Vector3(c.x - s.x * 0.4, c.y + s.y * 0.05, c.z), s, box.min, box.max),
            rarm: anchor(new THREE.Vector3(c.x + s.x * 0.4, c.y + s.y * 0.05, c.z), s, box.min, box.max),
            lleg: anchor(new THREE.Vector3(c.x - s.x * 0.15, box.min.y + s.y * 0.2, c.z), s, box.min, box.max),
            rleg: anchor(new THREE.Vector3(c.x + s.x * 0.15, box.min.y + s.y * 0.2, c.z), s, box.min, box.max)
        };
    }

    computeEndpoints() {
        const torso = this.limbAnchors.torso;
        if (!torso) return;
        ['larm', 'rarm', 'lleg', 'rleg'].forEach(key => {
            const a = this.limbAnchors[key];
            if (!a) return;
            const axis = a.size.x >= a.size.y && a.size.x >= a.size.z ? 'x' : (a.size.y >= a.size.z ? 'y' : 'z');
            const loEnd = a.center.clone(); loEnd[axis] = a.min[axis];
            const hiEnd = a.center.clone(); hiEnd[axis] = a.max[axis];
            if (loEnd.distanceTo(torso.center) > hiEnd.distanceTo(torso.center)) {
                a.end = loEnd; a.root = hiEnd;
            } else {
                a.end = hiEnd; a.root = loEnd;
            }
        });
    }

    attachModel(model, cat) {
        const textureLoader = new THREE.TextureLoader();
        let texture = null;
        if (this.textureFile) {
            const url = URL.createObjectURL(this.textureFile);
            texture = textureLoader.load(url);
            texture.colorSpace = THREE.SRGBColorSpace;
        }

        model.traverse(child => {
            if (child.isMesh) {
                child.castShadow = true;
                child.receiveShadow = true;
                if (!child.material || (Array.isArray(child.material) && child.material.length === 0)) {
                    child.material = makeDefaultMaterial();
                }
                if (texture) {
                    const mats = Array.isArray(child.material) ? child.material : [child.material];
                    mats.forEach(m => {
                        m.map = texture;
                        m.needsUpdate = true;
                    });
                }
            }
        });

        const size = new THREE.Box3().setFromObject(model).getSize(new THREE.Vector3());

        const pivot = new THREE.Group();
        pivot.userData.baseScale = 1;
        pivot.userData.lift = cat === 'face' ? 0 : (size.y / 2);
        pivot.add(model);

        this.computeLimbAnchors();
        this.viewer.avatarGroup.add(pivot);
        this.pivot = pivot;
    }

    pointerToNDC(e) {
        const rect = this.canvas.getBoundingClientRect();
        return new THREE.Vector2(
            ((e.clientX - rect.left) / rect.width) * 2 - 1,
            -((e.clientY - rect.top) / rect.height) * 2 + 1
        );
    }

    hitsModel(e) {
        const ndc = this.pointerToNDC(e);
        raycaster.setFromCamera(ndc, this.viewer.camera);
        if (raycaster.intersectObject(this.pivot, true).length) return true;

        const center = this.pivot.getWorldPosition(new THREE.Vector3()).project(this.viewer.camera);
        const rect = this.canvas.getBoundingClientRect();
        const sx = (center.x * 0.5 + 0.5) * rect.width;
        const sy = (-center.y * 0.5 + 0.5) * rect.height;
        const px = e.clientX - rect.left;
        const py = e.clientY - rect.top;
        return Math.hypot(px - sx, py - sy) <= HIT_RADIUS;
    }

    hitAvatarPart() {
        const parts = Object.values(this.limbParts || {});
        if (!parts.length) return null;
        const hits = raycaster.intersectObjects(parts, true);
        for (const hit of hits) {
            let obj = hit.object;
            while (obj && !obj.userData.limbKey) obj = obj.parent;
            if (obj && obj.userData.limbKey) {
                const norm = hit.face ? hit.face.normal.clone().transformDirection(hit.object.matrixWorld) : null;
                return { limb: obj.userData.limbKey, point: hit.point, normal: norm, distance: hit.distance };
            }
        }
        return null;
    }

    placeOnPart(hit) {
        if (!hit) return false;
        const group = this.viewer.avatarGroup;
        const inv = new THREE.Matrix4().copy(group.matrixWorld).invert();
        const local = hit.point.clone().applyMatrix4(inv);
        const limb = this.limbAnchors[hit.limb];
        if (!limb) return false;

        let surface = 'top';
        if (hit.normal) {
            const n = hit.normal.clone().transformDirection(inv);
            const ax = Math.abs(n.x), ay = Math.abs(n.y), az = Math.abs(n.z);
            if (ay >= ax && ay >= az) {
                if (n.y < 0) {
                    if (hit.limb === 'larm' || hit.limb === 'rarm') surface = 'hand';
                    else if (hit.limb === 'lleg' || hit.limb === 'rleg') surface = 'foot';
                }
            } else if (az >= ax && n.z > 0) {
                surface = 'front';
            }
        }

        const snap = hit.limb + '-' + surface;
        if (SNAPS[snap] && this.snap !== snap) {
            this.snap = snap;
            this.setSnapValue(snap);
        }

        const t = this.getTransform();
        const base = this.getAttachmentPosition(this.snapValue);
        const lift = this.snapValue === 'none' ? 0 : (this.pivot.userData.lift || 0);
        const targetY = surface === 'top' ? limb.max.y : local.y;
        t.position.x = local.x - base.x;
        t.position.y = targetY - base.y - lift * t.scale.y;
        t.position.z = local.z - base.z;
        this.applyTransform();
        const liftY = this.resolveCollision();
        if (liftY > 0) t.position.y += liftY;
        this.syncInputs();
        return true;
    }

    handlePointerDown(e) {
        if (!this.viewer || !this.pivot || !this.hitsModel(e)) return;
        e.stopImmediatePropagation();

        const center = this.pivot.getWorldPosition(new THREE.Vector3());
        const normal = this.viewer.camera.getWorldDirection(new THREE.Vector3());
        dragPlane.setFromNormalAndCoplanarPoint(normal, center);

        const ndc = this.pointerToNDC(e);
        raycaster.setFromCamera(ndc, this.viewer.camera);
        const hit = new THREE.Vector3();
        if (!intersectRayPlane(raycaster.ray, dragPlane, hit)) return;

        const group = this.viewer.avatarGroup;
        group.updateWorldMatrix(true, true);
        const inv = new THREE.Matrix4().copy(group.matrixWorld).invert();
        const localHit = hit.clone().applyMatrix4(inv);

        this.dragState = {
            pointerId: e.pointerId,
            grabOffset: localHit.sub(this.pivot.position.clone())
        };
    }

    resolveCollision() {
        if (!this.pivot || this.snapValue === 'none') return 0;
        const group = this.viewer.avatarGroup;
        group.updateWorldMatrix(true, true);
        const inv = new THREE.Matrix4().copy(group.matrixWorld).invert();
        const box = new THREE.Box3().setFromObject(this.pivot).applyMatrix4(inv);
        if (box.isEmpty()) return 0;

        const snap = this.snapValue;
        let liftY = 0;
        for (const key of Object.keys(this.limbAnchors)) {
            if (snap === key + '-hand' || snap === key + '-foot' || snap === key + '-front') continue;
            const a = this.limbAnchors[key];
            if (!a) continue;
            if (box.max.x <= a.min.x || box.min.x >= a.max.x) continue;
            if (box.max.z <= a.min.z || box.min.z >= a.max.z) continue;
            if (box.min.y < a.max.y) {
                liftY = Math.max(liftY, a.max.y - box.min.y);
            }
        }
        if (liftY > 0) {
            this.pivot.position.y += liftY;
        }
        return liftY;
    }

    handlePointerMove(e) {
        const drag = this.dragState;
        if (!this.viewer || !this.pivot || !drag || drag.pointerId !== e.pointerId) return;

        const ndc = this.pointerToNDC(e);
        raycaster.setFromCamera(ndc, this.viewer.camera);

        const modelHit = raycaster.intersectObject(this.pivot, true)[0];
        const partHit = this.hitAvatarPart();
        if (this.snapValue !== 'none' && (!modelHit || (partHit && partHit.distance < modelHit.distance))) {
            if (this.placeOnPart(partHit)) return;
        }

        const center = this.pivot.getWorldPosition(new THREE.Vector3());
        const normal = this.viewer.camera.getWorldDirection(new THREE.Vector3());
        dragPlane.setFromNormalAndCoplanarPoint(normal, center);

        const hit = new THREE.Vector3();
        if (!intersectRayPlane(raycaster.ray, dragPlane, hit)) return;

        const group = this.viewer.avatarGroup;
        group.updateWorldMatrix(true, true);
        const inv = new THREE.Matrix4().copy(group.matrixWorld).invert();
        const localTarget = hit.clone().applyMatrix4(inv).sub(drag.grabOffset);

        const t = this.getTransform();
        const base = this.getAttachmentPosition(this.snapValue);
        const lift = this.snapValue === 'none' ? 0 : (this.pivot.userData.lift || 0);
        t.position.x = localTarget.x - base.x;
        t.position.y = localTarget.y - base.y - lift * t.scale.y;
        t.position.z = localTarget.z - base.z;
        this.applyTransform();
        const liftY = this.resolveCollision();
        if (liftY > 0) t.position.y += liftY;
        this.syncInputs();
    }

    handlePointerUp(e) {
        const drag = this.dragState;
        if (drag && drag.pointerId === e.pointerId) {
            this.dragState = null;
            this.scheduleRefresh();
        }
    }

    attachDragListeners() {
        this.container = this.viewEl;
        this.container.addEventListener('pointerdown', this.onViewPointerDown, true);
        window.addEventListener('pointermove', this.onWindowPointerMove);
        window.addEventListener('pointerup', this.onWindowPointerUp);
        window.addEventListener('pointercancel', this.onWindowPointerUp);
    }

    removeDragListeners() {
        if (this.container) {
            this.container.removeEventListener('pointerdown', this.onViewPointerDown, true);
            this.container = null;
        }
        window.removeEventListener('pointermove', this.onWindowPointerMove);
        window.removeEventListener('pointerup', this.onWindowPointerUp);
        window.removeEventListener('pointercancel', this.onWindowPointerUp);
    }

    applyFromInputs() {
        this.transform = this.readInputs();
        this.applyTransform();
        this.scheduleRefresh();
    }

    setSnap() {
        const sel = document.getElementById('crt3dSnap');
        if (sel) this.snap = sel.value;
        const t = this.getTransform();
        t.position.x = 0;
        t.position.y = 0;
        t.position.z = 0;
        this.syncInputs();
        this.applyTransform();
        this.scheduleRefresh();
    }

    reset() {
        this.transform = null;
        this.snap = null;
        this.syncInputs();
        this.setSnapValue(this.snapValue);
        this.applyTransform();
        this.scheduleRefresh();
    }

    getTransformData() {
        const t = this.getTransform();
        return {
            position: { x: t.position.x, y: t.position.y, z: t.position.z },
            rotation: { x: t.rotation.x, y: t.rotation.y, z: t.rotation.z },
            scale: { x: t.scale.x, y: t.scale.y, z: t.scale.z },
            weld: this.weld,
            snap: this.snapValue
        };
    }

    close() {
        this.dragState = null;
        this.removeDragListeners();
        if (this.viewer) {
            this.viewer.destroy();
            this.viewer = null;
            this.scheduleRefresh();
        }
        this.pivot = null;
        this.limbAnchors = {};
        this.limbParts = {};
        const modal = this.modal;
        if (modal) modal.classList.remove('active');
    }

    async open() {
        const file = this.modelFile;
        const modal = this.modal;
        const viewEl = this.viewEl;
        if (!file || !modal || !viewEl) return;

        this.close();
        modal.classList.add('active');
        viewEl.innerHTML = '';

        this.viewer = new AvatarViewer(viewEl, getCurrentUserId());
        this.viewer.autoRotate = false;
        const viewer = this.viewer;

        try {
            await viewer.load();
            if (this.viewer !== viewer) return;

            const model = await loadModelFile(file);
            if (!model) throw new Error('Failed to load model');
            if (this.viewer !== viewer || !modal.classList.contains('active')) return;

            this.attachModel(model, this.cat);
            this.syncInputs();
            this.setSnapValue(this.snapValue);
            this.applyTransform();
            this.attachDragListeners();
        } catch (e) {
            if (this.viewer === viewer) {
                this.close();
                if (this.showError) {
                    this.showError('Could not load the 3D model. Make sure it is a valid GLB or OBJ file.');
                }
            }
        }
    }
}

const preview = new ModelPreview({ showError: window.createShowError || null });

const initialCatTab = document.querySelector('.crtshoptabs .tabbtn.active');
if (initialCatTab && initialCatTab.dataset.cat) preview.setCategory(initialCatTab.dataset.cat);

window.openCreate3DPreview = function() { preview.open(); };
window.closeCreate3DPreview = function() { preview.close(); };
window.applyCreateTransform = function() { preview.applyFromInputs(); };
window.setCreateSnap = function() { preview.setSnap(); };
window.resetCreateTransform = function() { preview.reset(); };
window.setCreateModelFile = function(file) { preview.setModel(file); };
window.setCreateTextureFile = function(file) { preview.setTexture(file); };
window.setCreateCategory = function(cat) { preview.setCategory(cat); };
window.getCreateModelFile = function() { return preview.modelFile; };
window.refreshCreatePreview = function() { preview.refreshPreview(); };
window.getCreateModelTransformData = function() { return preview.getTransformData(); };

document.addEventListener('htmx:beforeTransition', function() {
    preview.close();
});
