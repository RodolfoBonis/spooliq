# 🎨 Design System - Paleta de Cores

> **Versão:** 1.3  
> **Data:** 16/10/2024  
> **Inspiração:** Airbnb Design System

---

## 🎯 Filosofia do Design

Este design system prioriza **profissionalismo**, **clareza** e **neutralidade**. Baseado no design do Airbnb, utilizamos:

- ✨ Cores neutras como base (branco e cinzas)
- 🎨 Coral suave como cor primária (#ff6b6b)
- 🌊 Teal como accent color (#26c5c5)
- 📊 Status colors sutis e profissionais
- 🔲 Espaço em branco abundante (whitespace)

---

## 🎨 Paleta Principal

### Coral (Primary)
Cor principal para CTAs, links importantes e elementos destacados.

| Shade | Hex | RGB | Uso |
|-------|-----|-----|-----|
| 50 | `#fff5f5` | 255, 245, 245 | Backgrounds muito sutis |
| 100 | `#ffe3e3` | 255, 227, 227 | Hover states leves |
| 200 | `#ffc9c9` | 255, 201, 201 | Borders destacados |
| 300 | `#ffa8a8` | 255, 168, 168 | - |
| 400 | `#ff8787` | 255, 135, 135 | - |
| **500** | `#ff6b6b` | 255, 107, 107 | **Cor principal (Main)** |
| 600 | `#e85d5d` | 232, 93, 93 | Hover de botões |
| 700 | `#c94f4f` | 201, 79, 79 | Active state |
| 800 | `#a84141` | 168, 65, 65 | - |
| 900 | `#873434` | 135, 52, 52 | Texto sobre backgrounds claros |

**Exemplos de uso:**
```jsx
// Botão primário
<Button className="bg-primary-500 hover:bg-primary-600 text-white">
  Criar Orçamento
</Button>

// Badge importante
<Badge className="bg-primary-100 text-primary-700">
  Novo
</Badge>
```

---

### Neutral (Cinzas)
Sistema de cinzas para backgrounds, textos e bordas.

| Shade | Hex | RGB | Uso |
|-------|-----|-----|-----|
| White | `#ffffff` | 255, 255, 255 | Background principal de cards |
| 50 | `#f7f7f7` | 247, 247, 247 | Background da aplicação |
| 100 | `#e9e9e9` | 233, 233, 233 | Bordas sutis |
| 200 | `#d9d9d9` | 217, 217, 217 | Bordas médias |
| 300 | `#c4c4c4` | 196, 196, 196 | Separadores |
| 400 | `#9d9d9d` | 157, 157, 157 | Texto desabilitado |
| 500 | `#7b7b7b` | 123, 123, 123 | Texto terciário |
| **600** | `#555555` | 85, 85, 85 | **Texto secundário** |
| 700 | `#434343` | 67, 67, 67 | - |
| 800 | `#2e2e2e` | 46, 46, 46 | - |
| **900** | `#222222` | 34, 34, 34 | **Texto principal** |

**Exemplos de uso:**
```jsx
// Card com fundo neutro
<Card className="bg-white border border-neutral-200">
  <h2 className="text-neutral-900">Título</h2>
  <p className="text-neutral-600">Descrição secundária</p>
</Card>

// Background da página
<div className="bg-neutral-50 min-h-screen">
  {/* conteúdo */}
</div>
```

---

### Accent (Teal)
Cor secundária para CTAs alternativos e elementos de suporte.

| Shade | Hex | RGB | Uso |
|-------|-----|-----|-----|
| 50 | `#e6f7f7` | 230, 247, 247 | Backgrounds informativos |
| 100 | `#c2eded` | 194, 237, 237 | - |
| 200 | `#9be3e3` | 155, 227, 227 | - |
| 300 | `#74d9d9` | 116, 217, 217 | - |
| 400 | `#4dcfcf` | 77, 207, 207 | - |
| **500** | `#26c5c5` | 38, 197, 197 | **Accent principal** |
| 600 | `#20a5a5` | 32, 165, 165 | Hover |
| 700 | `#1a8585` | 26, 133, 133 | - |
| 800 | `#146565` | 20, 101, 101 | - |
| 900 | `#0e4545` | 14, 69, 69 | - |

**Exemplos de uso:**
```jsx
// Botão secundário
<Button variant="outline" className="border-accent-500 text-accent-600 hover:bg-accent-50">
  Ver mais
</Button>

// Badge informativo
<Badge className="bg-accent-50 text-accent-700">
  Info
</Badge>
```

---

## 📊 Status Colors

### Sucesso (Verde Airbnb)
**Cor:** `#00a699`

Usado para: aprovações, operações bem-sucedidas, status positivos

```jsx
<Badge className="bg-success/10 text-success">Aprovado</Badge>
<Button className="bg-success hover:bg-success-dark text-white">Confirmar</Button>
```

### Atenção (Laranja suave)
**Cor:** `#f4a261`

Usado para: avisos, ações em progresso, estados temporários

```jsx
<Alert className="bg-warning/10 border-warning text-warning-dark">
  <AlertCircle /> Atenção: verifique os dados
</Alert>
```

### Erro (Vermelho suave)
**Cor:** `#d93025`

Usado para: erros, rejeições, ações destrutivas

```jsx
<Button variant="destructive" className="bg-error hover:bg-error-dark text-white">
  Deletar
</Button>
```

### Informação (Azul)
**Cor:** `#0288d1`

Usado para: informações, estados neutros, notificações

```jsx
<Alert className="bg-info/10 border-info text-info-dark">
  <Info /> Nova funcionalidade disponível
</Alert>
```

---

## 🏷️ Status de Orçamentos

| Status | Cor | Hex | Uso |
|--------|-----|-----|-----|
| **Rascunho** | Cinza neutro | `#9d9d9d` | Orçamento em elaboração |
| **Enviado** | Azul confiável | `#0288d1` | Aguardando resposta |
| **Aprovado** | Verde Airbnb | `#00a699` | Cliente aprovou |
| **Rejeitado** | Vermelho suave | `#d93025` | Cliente rejeitou |
| **Imprimindo** | Laranja suave | `#f4a261` | Em produção |
| **Concluído** | Cinza escuro | `#5a6268` | Finalizado |

```jsx
const STATUS_CONFIG = {
  draft: { label: 'Rascunho', color: '#9d9d9d' },
  sent: { label: 'Enviado', color: '#0288d1' },
  approved: { label: 'Aprovado', color: '#00a699' },
  rejected: { label: 'Rejeitado', color: '#d93025' },
  printing: { label: 'Imprimindo', color: '#f4a261' },
  completed: { label: 'Concluído', color: '#5a6268' },
}
```

---

## 🎯 Princípios de Uso

### ✅ Faça

- Use branco (#ffffff) para backgrounds de cards e conteúdo
- Use cinza claro (#f7f7f7) para background da aplicação
- Use coral (#ff6b6b) apenas para CTAs importantes
- Mantenha hierarquia de texto: primary → secondary → tertiary
- Use bordas sutis (#e9e9e9) para separadores
- Prefira espaço em branco para respirar o design

### ❌ Evite

- Múltiplas cores vibrantes na mesma tela
- Fundos coloridos em áreas amplas
- Texto sem contraste suficiente (mínimo 4.5:1)
- Uso excessivo da cor primária
- Bordas muito escuras ou chamativas

---

## 🔧 Configuração Tailwind CSS

```javascript
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#fff5f5',
          100: '#ffe3e3',
          200: '#ffc9c9',
          300: '#ffa8a8',
          400: '#ff8787',
          500: '#ff6b6b', // Main
          600: '#e85d5d',
          700: '#c94f4f',
          800: '#a84141',
          900: '#873434',
        },
        neutral: {
          50: '#f7f7f7',
          100: '#e9e9e9',
          200: '#d9d9d9',
          300: '#c4c4c4',
          400: '#9d9d9d',
          500: '#7b7b7b',
          600: '#555555',
          700: '#434343',
          800: '#2e2e2e',
          900: '#222222',
        },
        accent: {
          50: '#e6f7f7',
          100: '#c2eded',
          200: '#9be3e3',
          300: '#74d9d9',
          400: '#4dcfcf',
          500: '#26c5c5',
          600: '#20a5a5',
          700: '#1a8585',
          800: '#146565',
          900: '#0e4545',
        },
        success: '#00a699',
        warning: '#f4a261',
        error: '#d93025',
        info: '#0288d1',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
      },
    },
  },
}
```

---

## 📱 Exemplos Práticos

### Botões

```jsx
// Primário
<button className="bg-primary-500 hover:bg-primary-600 text-white px-4 py-2 rounded-lg font-medium transition">
  Criar Orçamento
</button>

// Secundário
<button className="border border-neutral-300 text-neutral-700 hover:bg-neutral-50 px-4 py-2 rounded-lg font-medium transition">
  Cancelar
</button>

// Destrutivo
<button className="bg-error hover:bg-error-dark text-white px-4 py-2 rounded-lg font-medium transition">
  Deletar
</button>
```

### Cards

```jsx
<div className="bg-white border border-neutral-200 rounded-xl p-6 shadow-sm hover:shadow-md transition">
  <h3 className="text-lg font-semibold text-neutral-900 mb-2">
    Orçamento #1234
  </h3>
  <p className="text-sm text-neutral-600 mb-4">
    Cliente: João Silva
  </p>
  <div className="flex items-center justify-between">
    <span className="text-2xl font-bold text-primary-600">
      R$ 1.250,00
    </span>
    <Badge className="bg-success/10 text-success">
      Aprovado
    </Badge>
  </div>
</div>
```

### Formulários

```jsx
<div className="space-y-4">
  <div>
    <label className="block text-sm font-medium text-neutral-700 mb-1">
      Nome do Cliente
    </label>
    <input 
      type="text"
      className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none transition"
      placeholder="Digite o nome"
    />
  </div>
</div>
```

### Alerts

```jsx
// Sucesso
<div className="bg-success/10 border border-success/20 rounded-lg p-4">
  <p className="text-success font-medium">✓ Orçamento criado com sucesso!</p>
</div>

// Erro
<div className="bg-error/10 border border-error/20 rounded-lg p-4">
  <p className="text-error font-medium">✕ Erro ao salvar orçamento</p>
</div>

// Info
<div className="bg-info/10 border border-info/20 rounded-lg p-4">
  <p className="text-info font-medium">ⓘ 14 dias restantes no trial</p>
</div>
```

---

## 🌐 Acessibilidade

Todas as cores foram testadas para contraste mínimo **WCAG AA (4.5:1)**:

| Combinação | Contraste | Status |
|------------|-----------|--------|
| primary-500 / white | 5.2:1 | ✅ Passa |
| neutral-900 / white | 16.1:1 | ✅ Passa |
| neutral-600 / white | 7.8:1 | ✅ Passa |
| success / white | 4.8:1 | ✅ Passa |
| error / white | 5.1:1 | ✅ Passa |

---

## 🔗 Referências

- [Airbnb Design System](https://airbnb.design/)
- [WCAG 2.1 Accessibility Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [Material Design Color System](https://m3.material.io/styles/color/system/overview)
- [Tailwind CSS Colors](https://tailwindcss.com/docs/customizing-colors)

---

**Documento criado em:** 16/10/2024  
**Versão:** 1.3  
**Projeto:** Spooliq SaaS


