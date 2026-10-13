import * as THREE from 'three';
import { OBJLoader } from 'three/addons/loaders/OBJLoader.js';

window.THREE = THREE;

const ROTATION_SPEED = 0.008;
const AUTO_ROTATE_SPEED = 0.005;
const MIN_PHI = -1.45;
const MAX_PHI = 1.45;
const MIN_RADIUS = 2.2;
const MAX_RADIUS = 12.0;

const clamp = (val, min, max) => Math.max(min, Math.min(max, val));

export class AvatarViewer {
    constructor(container, userId) {
        this.container = container;
        this.userId = userId;

        this.target = new THREE.Vector3(0, 0.8, 0);
        this.spherical = {
            radius: 6.8,
            theta: -0.35,
            phi: 0.28
        };
        this.targetRadius = 6.8;

        this.isDragging = false;
        this.previousPointerPosition = { x: 0, y: 0 };
        this.initialPinchDistance = null;
        this.animationId = null;
        this.spinnerEl = null;
        this.autoRotate = true;

        this.initScene();
        this.initRenderer();
        this.initLights();
        this.initControls();
        this.initResizeObserver();
        this.showSpinner();
    }

    showSpinner() {
        this.spinnerEl = document.createElement('div');
        this.spinnerEl.className = 'musld';
        this.spinnerEl.style.position = 'absolute';
        this.spinnerEl.style.top = '50%';
        this.spinnerEl.style.left = '50%';
        this.spinnerEl.style.transform = 'translate(-50%, -50%)';
        this.spinnerEl.style.zIndex = '10';
        this.spinnerEl.style.pointerEvents = 'none';
        this.spinnerEl.style.margin = '0';
        this.spinnerEl.style.padding = '0';
        this.spinnerEl.innerHTML = `
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
        this.container.appendChild(this.spinnerEl);
    }

    hideSpinner() {
        if (this.spinnerEl && this.spinnerEl.parentNode) {
            this.spinnerEl.parentNode.removeChild(this.spinnerEl);
            this.spinnerEl = null;
        }
    }

    initScene() {
        this.scene = new THREE.Scene();
        const width = this.container.clientWidth || 300;
        const height = this.container.clientHeight || 340;

        this.camera = new THREE.PerspectiveCamera(60, width / height, 0.1, 1000);
        this.updateCameraPosition();

        this.avatarGroup = new THREE.Group();
        this.scene.add(this.avatarGroup);

        const shadowPlaneGeo = new THREE.PlaneGeometry(12, 12);
        const shadowPlaneMat = new THREE.ShadowMaterial({ opacity: 0.15 });
        this.shadowPlane = new THREE.Mesh(shadowPlaneGeo, shadowPlaneMat);
        this.shadowPlane.rotation.x = -Math.PI / 2;
        this.shadowPlane.position.y = -2.2;
        this.shadowPlane.receiveShadow = true;
        this.scene.add(this.shadowPlane);
    }

    initRenderer() {
        this.container.innerHTML = '';
        const width = this.container.clientWidth || 300;
        const height = this.container.clientHeight || 340;

        this.renderer = new THREE.WebGLRenderer({ alpha: true, antialias: true });
        this.renderer.setSize(width, height);
        this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));

        if (THREE.NeutralToneMapping) {
            this.renderer.toneMapping = THREE.NeutralToneMapping;
        } else if (THREE.LinearToneMapping) {
            this.renderer.toneMapping = THREE.LinearToneMapping;
        }
        this.renderer.toneMappingExposure = 1.0;

        if (THREE.PCFShadowMap) {
            this.renderer.shadowMap.enabled = true;
            this.renderer.shadowMap.type = THREE.PCFShadowMap;
        } else if (THREE.PCFSoftShadowMap) {
            this.renderer.shadowMap.enabled = true;
            this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        }

        this.container.appendChild(this.renderer.domElement);
    }

    initLights() {
        const hemiLight = new THREE.HemisphereLight(0xffffff, 0x888888, 0.8);
        this.scene.add(hemiLight);

        const keyLight = new THREE.DirectionalLight(0xffffff, 1.0);
        keyLight.position.set(4, 7, 5);
        if (keyLight.shadow) {
            keyLight.castShadow = true;
            keyLight.shadow.mapSize.width = 1024;
            keyLight.shadow.mapSize.height = 1024;
            keyLight.shadow.camera.near = 0.5;
            keyLight.shadow.camera.far = 18;
            keyLight.shadow.bias = -0.0005;
        }
        this.scene.add(keyLight);

        const fillLight = new THREE.DirectionalLight(0xffffff, 0.3);
        fillLight.position.set(-4, 3, 3);
        this.scene.add(fillLight);

        const pointLight = new THREE.PointLight(0xffffff, 0.25, 10);
        pointLight.position.set(0, 2, 2);
        this.scene.add(pointLight);
    }

    initControls() {
        this.activePointers = new Map();

        this.boundPointerDown = this.onPointerDown.bind(this);
        this.boundPointerMove = this.onPointerMove.bind(this);
        this.boundPointerUp = this.onPointerUp.bind(this);
        this.boundWheel = this.onWheel.bind(this);

        this.container.addEventListener('pointerdown', this.boundPointerDown);
        window.addEventListener('pointermove', this.boundPointerMove);
        window.addEventListener('pointerup', this.boundPointerUp);
        window.addEventListener('pointercancel', this.boundPointerUp);
        this.container.addEventListener('wheel', this.boundWheel, { passive: false });
    }

    removeEventListeners() {
        if (this.container) {
            this.container.removeEventListener('pointerdown', this.boundPointerDown);
            this.container.removeEventListener('wheel', this.boundWheel);
        }
        window.removeEventListener('pointermove', this.boundPointerMove);
        window.removeEventListener('pointerup', this.boundPointerUp);
        window.removeEventListener('pointercancel', this.boundPointerUp);
    }

    initResizeObserver() {
        if (window.ResizeObserver) {
            this.resizeObserver = new ResizeObserver(entries => {
                for (let entry of entries) {
                    const w = entry.contentRect.width;
                    const h = entry.contentRect.height;
                    if (w > 0 && h > 0 && this.renderer && this.camera) {
                        this.camera.aspect = w / h;
                        this.camera.updateProjectionMatrix();
                        this.renderer.setSize(w, h);
                    }
                }
            });
            this.resizeObserver.observe(this.container);
        }
    }

    rotateCamera(deltaX, deltaY) {
        this.spherical.theta -= deltaX * ROTATION_SPEED;
        this.spherical.phi += deltaY * ROTATION_SPEED;
        this.spherical.phi = clamp(this.spherical.phi, MIN_PHI, MAX_PHI);
    }

    zoomCamera(delta) {
        this.targetRadius += delta;
        this.targetRadius = clamp(this.targetRadius, MIN_RADIUS, MAX_RADIUS);
    }

    updateCameraPosition() {
        if (this.autoRotate && !this.isDragging && (!this.activePointers || this.activePointers.size === 0)) {
            this.spherical.theta += AUTO_ROTATE_SPEED;
        }

        this.spherical.radius += (this.targetRadius - this.spherical.radius) * 0.1;

        const x = this.target.x + this.spherical.radius * Math.sin(this.spherical.theta) * Math.cos(this.spherical.phi);
        const y = this.target.y + this.spherical.radius * Math.sin(this.spherical.phi);
        const z = this.target.z + this.spherical.radius * Math.cos(this.spherical.theta) * Math.cos(this.spherical.phi);

        this.camera.position.set(x, y, z);
        this.camera.lookAt(this.target);
    }

    onPointerDown(e) {
        this.activePointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
        if (this.activePointers.size === 1) {
            this.isDragging = true;
            this.previousPointerPosition = { x: e.clientX, y: e.clientY };
        } else if (this.activePointers.size === 2) {
            this.isDragging = false;
            const pts = Array.from(this.activePointers.values());
            this.initialPinchDistance = Math.hypot(pts[0].x - pts[1].x, pts[0].y - pts[1].y);
        }
    }

    onPointerMove(e) {
        if (!this.activePointers.has(e.pointerId)) return;
        this.activePointers.set(e.pointerId, { x: e.clientX, y: e.clientY });

        if (this.activePointers.size === 1 && this.isDragging) {
            const deltaX = e.clientX - this.previousPointerPosition.x;
            const deltaY = e.clientY - this.previousPointerPosition.y;

            this.rotateCamera(deltaX, deltaY);
            this.previousPointerPosition = { x: e.clientX, y: e.clientY };
        } else if (this.activePointers.size === 2 && this.initialPinchDistance !== null) {
            const pts = Array.from(this.activePointers.values());
            const currentDistance = Math.hypot(pts[0].x - pts[1].x, pts[0].y - pts[1].y);
            const deltaDistance = this.initialPinchDistance - currentDistance;
            this.zoomCamera(deltaDistance * 0.01);
            this.initialPinchDistance = currentDistance;
        }
    }

    onPointerUp(e) {
        this.activePointers.delete(e.pointerId);
        if (this.activePointers.size < 2) {
            this.initialPinchDistance = null;
        }
        if (this.activePointers.size === 1) {
            this.isDragging = true;
            const p = Array.from(this.activePointers.values())[0];
            this.previousPointerPosition = { x: p.x, y: p.y };
        } else if (this.activePointers.size === 0) {
            this.isDragging = false;
        }
    }

    onWheel(e) {
        e.preventDefault();
        this.zoomCamera(e.deltaY * 0.005);
    }

    async createHeadTexture(skinColor, faceUrl) {
        return new Promise(resolve => {
            const canvas = document.createElement('canvas');
            canvas.width = 512;
            canvas.height = 512;
            const ctx = canvas.getContext('2d');

            ctx.fillStyle = skinColor;
            ctx.fillRect(0, 0, 512, 512);

            const finishTexture = () => {
                const tex = new THREE.CanvasTexture(canvas);
                if (THREE.SRGBColorSpace) {
                    tex.colorSpace = THREE.SRGBColorSpace;
                }
                resolve(tex);
            };

            if (!faceUrl) {
                finishTexture();
                return;
            }

            const img = new Image();
            img.crossOrigin = 'anonymous';
            img.onload = () => {
                ctx.drawImage(img, 0, 0, 512, 512);
                finishTexture();
            };
            img.onerror = () => {
                finishTexture();
            };
            img.src = faceUrl;
        });
    }

    async fetchAvatarData() {
        try {
            const response = await fetch(`/api/v1/avatar/data/${this.userId}`);
            if (!response.ok) throw new Error('Failed to load avatar data');
            return await response.json();
        } catch {
            return null;
        }
    }

    async renderAvatar(data) {
        const faceId = data ? data.face_id : 0;
        const faceUrl = `/assets/faces/${faceId}.png`;

        const bodyParts = [
            { name: 'Head', file: 'Head.obj', color: data ? data.head_color : '#f3b700', isHead: true },
            { name: 'Torso', file: 'Torso.obj', color: data ? data.torso_color : '#c60000' },
            { name: 'LeftArm', file: 'LeftArm.obj', color: data ? data.larm_color : '#f3b700' },
            { name: 'RightArm', file: 'RightArm.obj', color: data ? data.rarm_color : '#f3b700' },
            { name: 'LeftLeg', file: 'LeftLeg.obj', color: data ? data.lleg_color : '#650013' },
            { name: 'RightLeg', file: 'RightLeg.obj', color: data ? data.rleg_color : '#650013' }
        ];

        const loader = new OBJLoader();

        const partPromises = bodyParts.map(async part => {
            try {
                const obj = await loader.loadAsync(`/assets/char/${part.file}`);
                return { part, obj };
            } catch {
                return { part, obj: null };
            }
        });

        const results = await Promise.all(partPromises);

        for (const { part, obj } of results) {
            if (!obj) continue;

            let material;
            if (part.isHead) {
                const headTex = await this.createHeadTexture(part.color, faceUrl);
                material = new THREE.MeshStandardMaterial({
                    map: headTex,
                    roughness: 0.6,
                    metalness: 0.0,
                    emissive: new THREE.Color(0xffffff),
                    emissiveMap: headTex,
                    emissiveIntensity: 0.2
                });
            } else {
                material = new THREE.MeshStandardMaterial({
                    color: new THREE.Color(part.color),
                    roughness: 0.6,
                    metalness: 0.0,
                    emissive: new THREE.Color(part.color),
                    emissiveIntensity: 0.2
                });
            }

            obj.traverse(child => {
                if (child.isMesh) {
                    child.material = material;
                    child.castShadow = true;
                    child.receiveShadow = true;
                }
            });

            this.avatarGroup.add(obj);
        }

        const box = new THREE.Box3().setFromObject(this.avatarGroup);
        if (!box.isEmpty()) {
            const center = box.getCenter(new THREE.Vector3());
            if (!isNaN(center.x) && !isNaN(center.y) && !isNaN(center.z)) {
                this.avatarGroup.position.x = -center.x;
                this.avatarGroup.position.y = -center.y + 0.1;
                this.avatarGroup.position.z = -center.z;
                this.shadowPlane.position.y = box.min.y - center.y + 0.1;
            }
        }
    }

    async load() {
        const data = await this.fetchAvatarData();
        await this.renderAvatar(data);
        this.hideSpinner();
        this.animate();
    }

    animate() {
        this.animationId = requestAnimationFrame(() => this.animate());
        this.updateCameraPosition();
        if (this.renderer && this.scene && this.camera) {
            this.renderer.render(this.scene, this.camera);
        }
    }

    disposeScene() {
        if (this.scene) {
            this.scene.traverse(obj => {
                if (obj.geometry) {
                    obj.geometry.dispose();
                }
                if (obj.material) {
                    if (Array.isArray(obj.material)) {
                        obj.material.forEach(m => {
                            if (m.map) m.map.dispose();
                            m.dispose();
                        });
                    } else {
                        if (obj.material.map) obj.material.map.dispose();
                        obj.material.dispose();
                    }
                }
            });
        }
        if (this.renderer) {
            if (this.renderer.domElement && this.renderer.domElement.parentNode) {
                this.renderer.domElement.parentNode.removeChild(this.renderer.domElement);
            }
            this.renderer.dispose();
        }
    }

    destroy() {
        if (this.animationId) {
            cancelAnimationFrame(this.animationId);
            this.animationId = null;
        }
        if (this.resizeObserver) {
            this.resizeObserver.disconnect();
            this.resizeObserver = null;
        }
        this.hideSpinner();
        this.removeEventListeners();
        this.disposeScene();
    }
}

let activeAvatarViewer = null;

window.init3DAvatar = function(userId) {
    if (activeAvatarViewer) {
        activeAvatarViewer.destroy();
        activeAvatarViewer = null;
    }

    const container = document.getElementById('avatar3d');
    if (!container) return;

    activeAvatarViewer = new AvatarViewer(container, userId);
    activeAvatarViewer.load();
};

const initProfile = () => {
    if (typeof window.updateProfileTabIndicator === 'function') {
        setTimeout(window.updateProfileTabIndicator, 50);
        setTimeout(window.updateProfileTabIndicator, 150);
        setTimeout(window.updateProfileTabIndicator, 350);
    }

    const container = document.getElementById('avatar3d');
    const userId = container ? container.dataset.userid : null;
    if (userId) {
        window.init3DAvatar(userId);
    }
};

initProfile();

document.addEventListener('htmx:afterSettle', initProfile);
document.addEventListener('htmx:beforeTransition', () => {
    if (activeAvatarViewer) {
        activeAvatarViewer.destroy();
        activeAvatarViewer = null;
    }
});
window.addEventListener('resize', () => {
    if (typeof window.updateProfileTabIndicator === 'function') {
        window.updateProfileTabIndicator();
    }
});