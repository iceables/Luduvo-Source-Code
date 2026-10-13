class GlobalColorPicker {
    constructor(container, options = {}) {
        this.container = typeof container === 'string' ? document.querySelector(container) : container;
        if (!this.container) return;

        this.options = Object.assign({
            width: 170,
            initialColor: '#F3B700',
            onChange: null,
            onInputEnd: null
        }, options);

        this.isSyncing = false;
        this.init();
    }

    init() {
        this.container.innerHTML = `
            <div class="glb-clrpick-wrap">
                <div class="glb-clrpick-wheel"></div>
                <div class="glb-clrpick-inputs">
                    <div class="glb-clrpick-line">
                        <span class="glb-clrpick-lbl">HEX</span>
                        <div class="setinbox glb-clrpick-inbox">
                            <input type="text" class="setin glb-clrpick-hex" maxlength="7" spellcheck="false" autocomplete="off">
                        </div>
                    </div>
                    <div class="glb-clrpick-line">
                        <span class="glb-clrpick-lbl">RGB</span>
                        <div class="glb-clrpick-rgbrw">
                            <div class="setinbox glb-clrpick-inbox-sm">
                                <input type="number" class="setin glb-clrpick-r" min="0" max="255" autocomplete="off">
                            </div>
                            <div class="setinbox glb-clrpick-inbox-sm">
                                <input type="number" class="setin glb-clrpick-g" min="0" max="255" autocomplete="off">
                            </div>
                            <div class="setinbox glb-clrpick-inbox-sm">
                                <input type="number" class="setin glb-clrpick-b" min="0" max="255" autocomplete="off">
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;

        this.wheelEl = this.container.querySelector('.glb-clrpick-wheel');
        this.hexInput = this.container.querySelector('.glb-clrpick-hex');
        this.rInput = this.container.querySelector('.glb-clrpick-r');
        this.gInput = this.container.querySelector('.glb-clrpick-g');
        this.bInput = this.container.querySelector('.glb-clrpick-b');

        if (typeof iro !== 'undefined') {
            this.iroPicker = new iro.ColorPicker(this.wheelEl, {
                width: this.options.width,
                color: this.options.initialColor,
                layout: [
                    { component: iro.ui.Wheel },
                    { component: iro.ui.Slider, options: { sliderType: 'value' } }
                ]
            });

            this.updateInputs(this.iroPicker.color);

            this.iroPicker.on('color:change', (color) => {
                if (this.isSyncing) return;
                this.updateInputs(color);
                if (typeof this.options.onChange === 'function') {
                    this.options.onChange(color.hexString, color.rgb);
                }
            });

            this.iroPicker.on('input:end', (color) => {
                if (this.isSyncing) return;
                if (typeof this.options.onInputEnd === 'function') {
                    this.options.onInputEnd(color.hexString, color.rgb);
                }
            });
        }

        this.bindEvents();
    }

    updateInputs(color) {
        this.isSyncing = true;
        if (this.hexInput) this.hexInput.value = color.hexString.toUpperCase();
        if (this.rInput) this.rInput.value = color.rgb.r;
        if (this.gInput) this.gInput.value = color.rgb.g;
        if (this.bInput) this.bInput.value = color.rgb.b;
        this.isSyncing = false;
    }

    setColor(colorVal) {
        if (!this.iroPicker) return;
        this.isSyncing = true;
        this.iroPicker.color.set(colorVal);
        this.updateInputs(this.iroPicker.color);
        this.isSyncing = false;
    }

    bindEvents() {
        if (this.hexInput) {
            this.hexInput.addEventListener('input', () => {
                if (this.isSyncing) return;
                let val = this.hexInput.value.trim();
                if (!val.startsWith('#')) val = '#' + val;
                if (/^#[0-9A-Fa-f]{6}$/.test(val)) {
                    this.isSyncing = true;
                    this.iroPicker.color.set(val);
                    if (this.rInput) this.rInput.value = this.iroPicker.color.rgb.r;
                    if (this.gInput) this.gInput.value = this.iroPicker.color.rgb.g;
                    if (this.bInput) this.bInput.value = this.iroPicker.color.rgb.b;
                    this.isSyncing = false;
                    if (typeof this.options.onChange === 'function') {
                        this.options.onChange(this.iroPicker.color.hexString, this.iroPicker.color.rgb);
                    }
                }
            });

            this.hexInput.addEventListener('change', () => {
                let val = this.hexInput.value.trim();
                if (!val.startsWith('#')) val = '#' + val;
                if (/^#[0-9A-Fa-f]{6}$/.test(val)) {
                    if (typeof this.options.onInputEnd === 'function') {
                        this.options.onInputEnd(this.iroPicker.color.hexString, this.iroPicker.color.rgb);
                    }
                }
            });
        }

        const onRgbInput = () => {
            if (this.isSyncing) return;
            const r = Math.min(255, Math.max(0, parseInt(this.rInput.value, 10) || 0));
            const g = Math.min(255, Math.max(0, parseInt(this.gInput.value, 10) || 0));
            const b = Math.min(255, Math.max(0, parseInt(this.bInput.value, 10) || 0));

            this.isSyncing = true;
            this.iroPicker.color.set({ r, g, b });
            if (this.hexInput) this.hexInput.value = this.iroPicker.color.hexString.toUpperCase();
            this.isSyncing = false;

            if (typeof this.options.onChange === 'function') {
                this.options.onChange(this.iroPicker.color.hexString, this.iroPicker.color.rgb);
            }
        };

        const onRgbChange = () => {
            if (typeof this.options.onInputEnd === 'function') {
                this.options.onInputEnd(this.iroPicker.color.hexString, this.iroPicker.color.rgb);
            }
        };

        [this.rInput, this.gInput, this.bInput].forEach(input => {
            if (input) {
                input.addEventListener('input', onRgbInput);
                input.addEventListener('change', onRgbChange);
            }
        });
    }
}

window.GlobalColorPicker = GlobalColorPicker;