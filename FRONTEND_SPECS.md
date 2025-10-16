# 🎨 Spooliq SaaS - Frontend Specifications

## 📋 Índice

1. [Visão Geral](#visão-geral)
2. [Stack Tecnológica](#stack-tecnológica)
3. [Design System](#design-system)
4. [Autenticação e Permissões](#autenticação-e-permissões)
5. [Estrutura de Páginas](#estrutura-de-páginas)
6. [API Endpoints](#api-endpoints)
7. [Fluxos de Usuário](#fluxos-de-usuário)
8. [Landing Page](#landing-page)
9. [Componentes Principais](#componentes-principais)
10. [Gerenciamento de Estado](#gerenciamento-de-estado)

---

## 🎯 Visão Geral

**Spooliq** é uma plataforma SaaS completa para gerenciamento de orçamentos de impressão 3D, com:
- Multi-tenancy (cada empresa tem seus próprios dados)
- Sistema de assinatura com trial de 14 dias
- Gerenciamento de clientes, filamentos, materiais e presets
- Geração automática de PDFs de orçamento
- Upload de arquivos para CDN
- Dashboard analítico

---

## 🛠 Stack Tecnológica

### Core
- **Framework**: Next.js 14+ (App Router)
- **Linguagem**: TypeScript
- **Styling**: Tailwind CSS + shadcn/ui
- **State Management**: Zustand + React Query
- **Forms**: React Hook Form + Zod
- **HTTP Client**: Axios

### Bibliotecas Essenciais
```json
{
  "dependencies": {
    "next": "^14.0.0",
    "react": "^18.2.0",
    "typescript": "^5.0.0",
    "tailwindcss": "^3.4.0",
    "@radix-ui/react-*": "latest", // Components do shadcn
    "zustand": "^4.4.0",
    "@tanstack/react-query": "^5.0.0",
    "axios": "^1.6.0",
    "react-hook-form": "^7.48.0",
    "zod": "^3.22.0",
    "@hookform/resolvers": "^3.3.0",
    "date-fns": "^2.30.0",
    "lucide-react": "^0.292.0",
    "recharts": "^2.10.0",
    "react-dropzone": "^14.2.0",
    "sonner": "^1.2.0" // Notifications
  }
}
```

---

## 🎨 Design System

### Paleta de Cores (Profissional e Neutra - Inspirada em Airbnb)

```css
:root {
  /* Primary - Coral/Vermelho suave (Airbnb-inspired) */
  --primary-50: #fff5f5;
  --primary-100: #ffe3e3;
  --primary-200: #ffc9c9;
  --primary-300: #ffa8a8;
  --primary-400: #ff8787;
  --primary-500: #ff6b6b; /* Main Coral */
  --primary-600: #e85d5d;
  --primary-700: #c94f4f;
  --primary-800: #a84141;
  --primary-900: #873434;

  /* Secondary - Cinza azulado escuro (profissional) */
  --secondary-50: #f8f9fa;
  --secondary-100: #f1f3f5;
  --secondary-200: #e9ecef;
  --secondary-300: #dee2e6;
  --secondary-400: #ced4da;
  --secondary-500: #6c757d;
  --secondary-600: #5a6268;
  --secondary-700: #484e53;
  --secondary-800: #343a40;
  --secondary-900: #212529;

  /* Neutral - Sistema de cinzas profissional */
  --neutral-white: #ffffff;
  --neutral-50: #f7f7f7;
  --neutral-100: #e9e9e9;
  --neutral-200: #d9d9d9;
  --neutral-300: #c4c4c4;
  --neutral-400: #9d9d9d;
  --neutral-500: #7b7b7b;
  --neutral-600: #555555;
  --neutral-700: #434343;
  --neutral-800: #2e2e2e;
  --neutral-900: #222222;

  /* Accent - Teal suave (call-to-action secundário) */
  --accent-50: #e6f7f7;
  --accent-100: #c2eded;
  --accent-200: #9be3e3;
  --accent-300: #74d9d9;
  --accent-400: #4dcfcf;
  --accent-500: #26c5c5;
  --accent-600: #20a5a5;
  --accent-700: #1a8585;
  --accent-800: #146565;
  --accent-900: #0e4545;

  /* Status Colors - Tons mais suaves e profissionais */
  --success-light: #d4edda;
  --success: #00a699;
  --success-dark: #008489;
  
  --warning-light: #fff3cd;
  --warning: #f4a261;
  --warning-dark: #e76f51;
  
  --error-light: #f8d7da;
  --error: #d93025;
  --error-dark: #b71c1c;
  
  --info-light: #d1ecf1;
  --info: #0288d1;
  --info-dark: #01579b;

  /* Budget Status Colors - Profissional */
  --status-draft: #9d9d9d;      /* Cinza neutro */
  --status-sent: #0288d1;        /* Azul confiável */
  --status-approved: #00a699;    /* Verde Airbnb */
  --status-rejected: #d93025;    /* Vermelho suave */
  --status-printing: #f4a261;    /* Laranja suave */
  --status-completed: #5a6268;   /* Cinza escuro elegante */

  /* Backgrounds - Tons neutros para aplicação */
  --bg-primary: #ffffff;
  --bg-secondary: #f7f7f7;
  --bg-tertiary: #e9e9e9;
  --bg-overlay: rgba(34, 34, 34, 0.6);

  /* Text Colors - Hierarquia visual clara */
  --text-primary: #222222;
  --text-secondary: #555555;
  --text-tertiary: #7b7b7b;
  --text-disabled: #9d9d9d;
  --text-inverse: #ffffff;

  /* Borders - Sutis e profissionais */
  --border-light: #e9e9e9;
  --border-medium: #d9d9d9;
  --border-dark: #c4c4c4;
}
```

### Guia de Uso das Cores

**Cores Primárias (Coral)**
- Usar para: Botões CTA principais, links importantes, elementos destacados
- Exemplos: "Criar Orçamento", "Salvar", "Confirmar", badges importantes
- Sempre com texto branco (#ffffff) por cima

**Cores Neutras (Cinzas)**
- Background principal: `--bg-primary` (#ffffff) - Fundo de cards e páginas
- Background secundário: `--bg-secondary` (#f7f7f7) - Fundo da aplicação, áreas secundárias
- Textos: Usar hierarquia `--text-primary` → `--text-secondary` → `--text-tertiary`
- Bordas: Sempre sutis, preferir `--border-light` (#e9e9e9) para separadores

**Cores de Accent (Teal)**
- Usar para: CTAs secundários, hover states, badges informativos
- Menos frequente que o coral, cria contraste interessante
- Bom para elementos de suporte como tooltips e notificações neutras

**Cores de Status**
- Verde (#00a699): Sucesso, aprovação, status positivo
- Azul (#0288d1): Informação, estados neutros, enviado
- Laranja (#f4a261): Atenção, em progresso, warning suave
- Vermelho (#d93025): Erro, rejeição, ações destrutivas
- Cinza (#9d9d9d): Rascunhos, desabilitado, neutro

**Princípios de Design**
- ✅ Priorizar branco e cinzas claros para backgrounds amplos
- ✅ Usar coral (#ff6b6b) com parcimônia para CTAs importantes
- ✅ Manter contraste mínimo de 4.5:1 para acessibilidade (WCAG AA)
- ✅ Evitar múltiplas cores chamativas na mesma tela
- ✅ Preferir espaço em branco (whitespace) para respirar o design

### Configuração Tailwind CSS

Para usar essas cores no Tailwind, adicione no `tailwind.config.js`:

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
          500: '#26c5c5', // Main
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

**Exemplos de uso:**
```jsx
// Botão primário
<button className="bg-primary-500 hover:bg-primary-600 text-white">
  Criar Orçamento
</button>

// Card com fundo neutro
<div className="bg-neutral-50 border border-neutral-200 rounded-lg">
  ...
</div>

// Badge de status
<span className="bg-success/10 text-success px-3 py-1 rounded-full">
  Aprovado
</span>
```

### Tipografia

```css
/* Font Family */
font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;

/* Scale */
--text-xs: 0.75rem;    /* 12px */
--text-sm: 0.875rem;   /* 14px */
--text-base: 1rem;     /* 16px */
--text-lg: 1.125rem;   /* 18px */
--text-xl: 1.25rem;    /* 20px */
--text-2xl: 1.5rem;    /* 24px */
--text-3xl: 1.875rem;  /* 30px */
--text-4xl: 2.25rem;   /* 36px */
--text-5xl: 3rem;      /* 48px */
```

### Espaçamento

```
4px, 8px, 12px, 16px, 20px, 24px, 32px, 40px, 48px, 64px, 80px, 96px
```

### Componentes Base (shadcn/ui)

**Instalar todos os componentes necessários:**

```bash
npx shadcn-ui@latest init
npx shadcn-ui@latest add button
npx shadcn-ui@latest add input
npx shadcn-ui@latest add card
npx shadcn-ui@latest add dialog
npx shadcn-ui@latest add dropdown-menu
npx shadcn-ui@latest add table
npx shadcn-ui@latest add tabs
npx shadcn-ui@latest add form
npx shadcn-ui@latest add select
npx shadcn-ui@latest add badge
npx shadcn-ui@latest add avatar
npx shadcn-ui@latest add alert
npx shadcn-ui@latest add toast
npx shadcn-ui@latest add skeleton
npx shadcn-ui@latest add sheet
npx shadcn-ui@latest add separator
npx shadcn-ui@latest add popover
npx shadcn-ui@latest add command
```

---

## 🔐 Autenticação e Permissões

### Keycloak Integration

**Environment Variables:**
```env
NEXT_PUBLIC_API_URL=http://localhost:8080/v1
NEXT_PUBLIC_KEYCLOAK_URL=https://auth.rodolfodebonis.com.br
NEXT_PUBLIC_KEYCLOAK_REALM=spooliq
NEXT_PUBLIC_KEYCLOAK_CLIENT_ID=spooliq
```

### Níveis de Permissão

| Role | Descrição | Acesso |
|------|-----------|--------|
| **PlatformAdmin** | Admin da plataforma | Tudo + gerenciar todas empresas |
| **Owner** | Dono da empresa | Tudo da empresa + gerenciar usuários e assinatura |
| **OrgAdmin** | Administrador | Tudo exceto gerenciar assinatura e deletar Owner |
| **User** | Usuário padrão | CRUD de recursos, sem acesso a configurações |

### Rotas Protegidas

```typescript
// lib/auth.ts
export const ROUTE_PERMISSIONS = {
  // Dashboard
  '/dashboard': ['PlatformAdmin', 'Owner', 'OrgAdmin', 'User'],
  
  // Budgets (Orçamentos)
  '/budgets': ['Owner', 'OrgAdmin', 'User'],
  '/budgets/new': ['Owner', 'OrgAdmin', 'User'],
  '/budgets/:id': ['Owner', 'OrgAdmin', 'User'],
  
  // Customers (Clientes)
  '/customers': ['Owner', 'OrgAdmin', 'User'],
  
  // Products (Filamentos, Materiais, Marcas)
  '/filaments': ['Owner', 'OrgAdmin', 'User'],
  '/materials': ['Owner', 'OrgAdmin', 'User'],
  '/brands': ['Owner', 'OrgAdmin', 'User'],
  
  // Presets (Máquinas, Energia, Custos)
  '/presets': ['Owner', 'OrgAdmin', 'User'],
  
  // Settings
  '/settings/company': ['Owner', 'OrgAdmin'],
  '/settings/branding': ['Owner', 'OrgAdmin'], // Personalização de cores do PDF
  '/settings/users': ['Owner', 'OrgAdmin'],
  '/settings/subscription': ['Owner'], // Apenas Owner
  
  // Platform Admin
  '/admin/companies': ['PlatformAdmin'],
  '/admin/subscriptions': ['PlatformAdmin'],
} as const;
```

### Auth Context/Store

```typescript
// stores/auth-store.ts
interface AuthState {
  user: User | null;
  token: string | null;
  organizationId: string | null;
  roles: string[];
  isAuthenticated: boolean;
  
  // Actions
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  refreshToken: () => Promise<void>;
  hasRole: (role: string | string[]) => boolean;
}

interface User {
  id: string;
  email: string;
  name: string;
  organizationId: string;
  roles: string[];
  avatar?: string;
}
```

---

## 📄 Estrutura de Páginas

```
app/
├── (auth)/                          # Layout sem sidebar
│   ├── login/
│   ├── register/
│   └── forgot-password/
│
├── (platform)/                      # Layout com sidebar (após login)
│   ├── dashboard/                   # Dashboard principal
│   │   └── page.tsx
│   │
│   ├── budgets/                     # Orçamentos
│   │   ├── page.tsx                # Lista
│   │   ├── new/page.tsx            # Criar novo
│   │   ├── [id]/page.tsx           # Detalhes
│   │   └── [id]/edit/page.tsx      # Editar
│   │
│   ├── customers/                   # Clientes
│   │   ├── page.tsx
│   │   ├── new/page.tsx
│   │   └── [id]/page.tsx
│   │
│   ├── catalog/                     # Catálogo de produtos
│   │   ├── filaments/
│   │   │   ├── page.tsx
│   │   │   └── new/page.tsx
│   │   ├── materials/
│   │   │   ├── page.tsx
│   │   │   └── new/page.tsx
│   │   └── brands/
│   │       ├── page.tsx
│   │       └── new/page.tsx
│   │
│   ├── presets/                     # Presets
│   │   ├── machines/page.tsx
│   │   ├── energy/page.tsx
│   │   └── costs/page.tsx
│   │
│   └── settings/                    # Configurações
│       ├── company/page.tsx         # Dados da empresa
│       ├── branding/page.tsx        # Personalização de cores do PDF
│       ├── users/page.tsx           # Gerenciar usuários
│       └── subscription/page.tsx    # Assinatura
│
├── (admin)/                         # Platform Admin
│   └── admin/
│       ├── companies/page.tsx
│       └── subscriptions/page.tsx
│
└── (marketing)/                     # Landing page
    ├── page.tsx                     # Home
    ├── features/page.tsx            # Features
    ├── pricing/page.tsx             # Planos
    └── contact/page.tsx             # Contato
```

---

## 🌐 API Endpoints

### Base URL
```
https://api.spooliq.com/v1
```

### ⚠️ IMPORTANTE: Convenções de Nomenclatura

**TODOS os campos da API usam snake_case, EXCETO:**
- `whatsapp` (SEM underscore - não é `whats_app`)

**Regras gerais:**
- Campos: `snake_case` (ex: `organization_id`, `created_at`)
- Valores monetários: **centavos** (ex: 10000 = R$ 100,00)
- Datas: **ISO 8601** (ex: "2024-10-15T10:30:00Z")
- IDs: **UUID v4**

### Authentication

#### 1. Register (Cadastro de nova empresa)
```typescript
POST /register
Body: {
  // User data
  name: string;                    // Nome completo do owner (min 3 chars)
  email: string;                   // Email válido
  password: string;                // Senha (min 8 caracteres)
  
  // Company data
  company_name: string;            // Nome da empresa (required)
  company_trade_name?: string;     // Nome fantasia (optional)
  company_document: string;        // CNPJ (required)
  company_phone: string;           // Telefone (required)
  
  // Address (all required)
  address: string;                 // Logradouro (required)
  address_number: string;          // Número (required)
  complement?: string;             // Complemento (optional)
  neighborhood: string;            // Bairro (required)
  city: string;                    // Cidade (required)
  state: string;                   // UF - 2 caracteres (required, ex: "SP")
  zip_code: string;                // CEP (required)
}
Response: {
  user_id: string;
  organization_id: string;
  trial_ends_at: string;           // ISO 8601 date
  message: string;
}
```

#### 2. Login
```typescript
POST /login
Body: {
  email: string;
  password: string;
}
Response: {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}
```

### Company (Empresa)

#### 1. Get Company Info
```typescript
GET /company/
Headers: { Authorization: Bearer {token} }
Response: {
  id: string;
  organization_id: string;
  name: string;
  trade_name?: string;
  document?: string;
  email?: string;
  phone?: string;
  whatsapp?: string;                  // ⚠️ SEM UNDERSCORE
  instagram?: string;
  website?: string;
  logo_url?: string;
  address?: string;
  city?: string;
  state?: string;
  zip_code?: string;
  
  // Subscription fields
  subscription_status: 'trial' | 'active' | 'overdue' | 'cancelled';
  is_platform_company: boolean;
  trial_ends_at?: string;             // ISO 8601 date
  subscription_started_at?: string;   // ISO 8601 date
  subscription_plan: 'basic' | 'pro' | 'enterprise';
  asaas_customer_id?: string;
  asaas_subscription_id?: string;
  last_payment_check?: string;        // ISO 8601 date
  next_payment_due?: string;          // ISO 8601 date
  
  created_at: string;
  updated_at: string;
}
```

#### 2. Update Company
```typescript
PUT /company/
Body: {
  name?: string;
  trade_name?: string;
  email?: string;
  phone?: string;
  whatsapp?: string;    // ⚠️ SEM UNDERSCORE
  instagram?: string;
  website?: string;
  address?: string;
  city?: string;
  state?: string;
  zip_code?: string;
}
Response: {
  message: string;
  company: CompanyEntity; // Same as GET response
}
```

#### 3. Upload Logo
```typescript
POST /company/logo
Content-Type: multipart/form-data
Body: FormData {
  file: File (PNG, JPG, JPEG - max 5MB)
}
Response: {
  message: string;
  logo_url: string;
}
```

#### 4. Get Company Branding
```typescript
GET /company/branding
Headers: { Authorization: Bearer {token} }
Response: {
  branding: {
    id: string;
    organization_id: string;
    template_name: string;
    
    // Header colors
    header_bg_color: string;      // #HEX
    header_text_color: string;    // #HEX
    
    // Primary colors
    primary_color: string;         // #HEX
    primary_text_color: string;    // #HEX
    
    // Secondary colors
    secondary_color: string;       // #HEX
    secondary_text_color: string;  // #HEX
    
    // Text colors
    title_color: string;           // #HEX
    body_text_color: string;       // #HEX
    
    // Accent colors
    accent_color: string;          // #HEX
    border_color: string;          // #HEX
    
    // Background colors
    background_color: string;      // #HEX
    table_header_bg_color: string; // #HEX
    table_row_alt_bg_color: string; // #HEX
    
    created_at: string;
    updated_at: string;
  }
}
```

#### 5. Update Company Branding
```typescript
PUT /company/branding
Body: {
  template_name?: string;          // "modern_pink", "corporate_blue", "tech_green", "elegant_purple", "custom"
  header_bg_color: string;         // #HEX (ex: "#ec4899")
  header_text_color: string;       // #HEX
  primary_color: string;           // #HEX
  primary_text_color: string;      // #HEX
  secondary_color: string;         // #HEX
  secondary_text_color: string;    // #HEX
  title_color: string;             // #HEX
  body_text_color: string;         // #HEX
  accent_color: string;            // #HEX
  border_color: string;            // #HEX
  background_color: string;        // #HEX
  table_header_bg_color: string;   // #HEX
  table_row_alt_bg_color: string;  // #HEX
}
Response: {
  message: string;
  branding: CompanyBrandingEntity; // Same as GET response
}
```

#### 6. List Branding Templates
```typescript
GET /company/branding/templates
Headers: { Authorization: Bearer {token} }
Response: {
  templates: Array<{
    name: string;                   // "modern_pink", "corporate_blue", etc.
    display_name: string;           // "Rosa Moderno", "Azul Corporativo", etc.
    description: string;            // Descrição do template
    colors: {
      template_name: string;
      header_bg_color: string;
      header_text_color: string;
      primary_color: string;
      primary_text_color: string;
      secondary_color: string;
      secondary_text_color: string;
      title_color: string;
      body_text_color: string;
      accent_color: string;
      border_color: string;
      background_color: string;
      table_header_bg_color: string;
      table_row_alt_bg_color: string;
    };
  }>;
}

// Templates disponíveis:
// 1. modern_pink - Rosa Moderno (padrão)
// 2. corporate_blue - Azul Corporativo
// 3. tech_green - Verde Tecnologia
// 4. elegant_purple - Roxo Elegante
```

### Budgets (Orçamentos)

#### 1. List Budgets
```typescript
GET /budgets?page=1&pageSize=20&status=draft
Response: {
  budgets: Array<{
    id: string;
    name: string;
    customer_id: string;
    customer: {
      id: string;
      name: string;
      email: string;
    };
    status: 'draft' | 'sent' | 'approved' | 'rejected' | 'printing' | 'completed';
    total_cost: number; // em centavos
    filament_cost: number;
    items_count: number;
    pdf_url?: string;
    created_at: string;
  }>;
  total: number;
  page: number;
  pageSize: number;
}
```

#### 2. Create Budget (✨ Multi-Filament Support)
```typescript
POST /v1/budgets
Body: {
  name: string;
  description?: string;
  customer_id: string;
  
  // Presets globais (aplicados a todos os items)
  machine_preset_id?: string;
  energy_preset_id?: string;
  
  // Flags de inclusão de custos
  include_energy_cost: boolean;
  include_waste_cost: boolean;
  
  // Informações comerciais
  delivery_days?: number;
  payment_terms?: string;
  notes?: string;
  
  // ITEMS = PRODUTOS (o que o cliente vê e compra)
  items: Array<{
    // === PRODUTO (Customer-facing) ===
    product_name: string;              // Nome do produto (ex: "Chaveiro Rosa/Branco")
    product_description?: string;      // Descrição detalhada
    product_quantity: number;          // Quantidade de UNIDADES (ex: 100 chaveiros)
    product_dimensions?: string;       // Dimensões (ex: "26×48×9 mm")
    
    // === TEMPO DE IMPRESSÃO (deste item) ===
    print_time_hours: number;          // Tempo para imprimir TODAS as unidades
    print_time_minutes: number;
    
    // === CUSTOS ADICIONAIS (opcionais) ===
    cost_preset_id?: string;           // Preset de custo específico para este item
    additional_labor_cost?: number;    // Custo extra de mão de obra em centavos (ex: pintura)
    additional_notes?: string;         // Observações sobre o item
    
    // === FILAMENTOS (Multi-filament support - AMS) ===
    filaments: Array<{
      filament_id: string;
      quantity: number;                // ⚠️ GRAMAS TOTAIS para este item (não por unidade!)
      order: number;                   // Ordem de aplicação (importante para AMS)
    }>;
    
    order: number;                     // Ordem de impressão (opcional)
  }>;
}

// ⚠️ IMPORTANTE: Quantidade de Filamento
// A quantidade é o TOTAL em gramas para o lote completo, NÃO por unidade!
// Exemplo: Para imprimir 100 chaveiros:
// - quantity: 2800.0  ✅ (total para os 100 chaveiros)
// - quantity: 28.0    ❌ (isso seria apenas 28g total!)
// 
// Por quê? Porque imprimir em lote tem economias de escala:
// - Menos desperdício (purge/prime)
// - Melhor aproveitamento do espaço
// - Otimização de camadas
// - 200 unidades ≠ 2× filamento de 100 unidades
```

#### 3. Get Budget (✨ Multi-Filament Response)
```typescript
GET /v1/budgets/:id
Response: {
  budget: {
    id: string;
    organization_id: string;
    name: string;
    description?: string;
    customer_id: string;
    status: 'draft' | 'sent' | 'approved' | 'rejected' | 'printing' | 'completed';
    
    // Presets
    machine_preset_id?: string;
    energy_preset_id?: string;
    
    // Flags
    include_energy_cost: boolean;
    include_waste_cost: boolean;
    
    // Custos calculados (SOMA de todos os items)
    filament_cost: number;        // em centavos
    waste_cost: number;           // em centavos
    energy_cost: number;          // em centavos
    labor_cost: number;           // em centavos
    total_cost: number;           // em centavos
    
    // Informações comerciais
    delivery_days?: number;
    payment_terms?: string;
    notes?: string;
    pdf_url?: string;
    owner_user_id: string;
    
    created_at: string;
    updated_at: string;
  };
  
  customer: {
    id: string;
    name: string;
    email: string;
    phone?: string;
    whatsapp?: string;
    document?: string;
  };
  
  items: Array<{
    id: string;
    budget_id: string;
    
    // === PRODUTO ===
    product_name: string;
    product_description?: string;
    product_quantity: number;          // unidades
    product_dimensions?: string;
    
    // === TEMPO DE IMPRESSÃO (deste item) ===
    print_time_hours: number;
    print_time_minutes: number;
    print_time_display: string;        // "5h30m" ou "45m"
    
    // === CUSTOS ADICIONAIS ===
    cost_preset_id?: string;
    additional_labor_cost?: number;    // centavos
    additional_notes?: string;
    
    // === CUSTOS CALCULADOS (deste item) ===
    filament_cost: number;             // centavos
    waste_cost: number;                // centavos
    energy_cost: number;               // centavos
    labor_cost: number;                // centavos
    item_total_cost: number;           // centavos (soma de todos acima)
    unit_price: number;                // centavos (item_total_cost ÷ product_quantity)
    
    // === FILAMENTOS (Multi-filament) ===
    filaments: Array<{
      filament_id: string;
      filament_name: string;
      brand_name: string;
      material_name: string;
      color: string;                   // Nome da cor ou hex
      quantity: number;                // gramas TOTAIS usadas neste item
      cost: number;                    // centavos (custo deste filamento)
      order: number;                   // ordem de aplicação
    }>;
    
    order: number;
    created_at: string;
    updated_at: string;
  }>;
  
  // === TEMPO TOTAL (calculado - soma de todos os items) ===
  total_print_time_hours: number;
  total_print_time_minutes: number;
  total_print_time_display: string;    // "14h15m"
}
```

#### 4. Update Budget Status
```typescript
PATCH /budgets/:id/status
Body: {
  status: 'sent' | 'approved' | 'rejected' | 'printing' | 'completed';
  notes?: string;
}
```

#### 5. Generate PDF
```typescript
GET /budgets/:id/pdf
Response: Binary (application/pdf)
// Download automático do PDF
```

#### 6. Delete Budget
```typescript
DELETE /budgets/:id
```

### Customers (Clientes)

#### 1. List Customers
```typescript
GET /customers?page=1&pageSize=20&search=João
Response: {
  customers: Array<{
    id: string;
    name: string;
    email: string;
    phone?: string;
    document?: string;
    address?: string;
    city?: string;
    state?: string;
    budgets_count: number;
    total_spent: number;
    created_at: string;
  }>;
  total: number;
}
```

#### 2. Create Customer
```typescript
POST /customers
Body: {
  name: string;
  email: string;
  phone?: string;
  whatsapp?: string;
  document?: string;    // CPF/CNPJ
  address?: string;
  city?: string;
  state?: string;
  zip_code?: string;
  notes?: string;
}
```

#### 3. Update Customer
```typescript
PUT /customers/:id
Body: {
  name?: string;
  email?: string;
  phone?: string;
  // ... outros campos
}
```

### Filaments (Filamentos)

#### 1. List Filaments
```typescript
GET /filaments?page=1&pageSize=20&brand_id=xxx&material_id=yyy
Response: {
  filaments: Array<{
    id: string;
    name: string;
    brand_id: string;
    brand_name: string;
    material_id: string;
    material_name: string;
    color_type: 'solid' | 'gradient' | 'duo' | 'rainbow';
    color_data: any;
    color_preview: string; // CSS para preview
    diameter: number;      // 1.75 ou 2.85
    price_per_kg: number;  // em centavos
    stock_quantity?: number;
    is_active: boolean;
    created_at: string;
  }>;
  total: number;
}
```

#### 2. Create Filament
```typescript
POST /filaments
Body: {
  name: string;
  brand_id: string;
  material_id: string;
  color_type: 'solid' | 'gradient' | 'duo' | 'rainbow';
  color_data: {
    // Solid
    color: string; // #RRGGBB
    
    // Gradient
    from: string;
    to: string;
    direction?: 'horizontal' | 'vertical' | 'diagonal';
    
    // Duo
    primary: string;
    secondary: string;
    ratio?: number;
    
    // Rainbow
    colors: string[];
    pattern?: string;
  };
  diameter: 1.75 | 2.85;
  price_per_kg: number;
  stock_quantity?: number;
  min_stock_alert?: number;
  description?: string;
}
```

### Brands (Marcas)

```typescript
GET /brands?page=1&pageSize=50
POST /brands
Body: {
  name: string;
  website?: string;
  description?: string;
}
PUT /brands/:id
DELETE /brands/:id
```

### Materials (Materiais)

```typescript
GET /materials?page=1&pageSize=50
POST /materials
Body: {
  name: string;              // PLA, ABS, PETG, TPU, etc.
  description?: string;
  properties?: {
    density?: number;        // g/cm³
    print_temp_min?: number; // °C
    print_temp_max?: number;
    bed_temp?: number;
  };
}
PUT /materials/:id
DELETE /materials/:id
```

### Presets

#### Machine Presets
```typescript
GET /presets/machines
POST /presets/machines
Body: {
  name: string;
  description?: string;
  waste_percentage: number; // AMS waste (ex: 15 para 15%)
  is_default?: boolean;
}
```

#### Energy Presets
```typescript
GET /presets/energy
POST /presets/energy
Body: {
  name: string;
  kwh_cost: number;          // Custo por kWh em centavos
  printer_power: number;     // Potência em Watts
  is_default?: boolean;
}
```

#### Cost Presets
```typescript
GET /presets/costs
POST /presets/costs
Body: {
  name: string;
  labor_cost_per_hour: number; // Custo de mão de obra/hora
  profit_margin?: number;       // Margem de lucro %
  is_default?: boolean;
}
```

### Users (Gerenciamento de usuários)

```typescript
GET /users              // Listar usuários da organização
POST /users             // Criar novo usuário (Owner/OrgAdmin)
Body: {
  name: string;
  email: string;
  password: string;
  role: 'OrgAdmin' | 'User';
}
PUT /users/:id          // Atualizar usuário
DELETE /users/:id       // Deletar usuário (apenas Owner)
```

### Admin (Platform Admin apenas)

```typescript
GET /admin/companies?page=1&status=active
GET /admin/companies/:organization_id
PATCH /admin/companies/:organization_id/status
Body: { status: 'active' | 'suspended' | 'cancelled' }

GET /admin/subscriptions
GET /admin/subscriptions/:organization_id
GET /admin/subscriptions/:organization_id/payments
```

---

## 🔄 Fluxos de Usuário

### 1. Onboarding (Novo usuário)

```
1. Landing Page
   ↓
2. Clique em "Começar agora" / "Criar conta"
   ↓
3. Página de Registro
   - Nome completo
   - Email
   - Senha
   - Nome da empresa
   - Telefone (opcional)
   ↓
4. Conta criada → Trial de 14 dias ativado
   ↓
5. Redirect para Dashboard
   ↓
6. Modal de Boas-vindas
   - "Bem-vindo ao Spooliq!"
   - Tour guiado (opcional)
   - Botão: "Configurar empresa"
   ↓
7. Configuração inicial (wizard)
   Step 1: Dados da empresa (logo, endereço)
   Step 2: Adicionar primeiro cliente
   Step 3: Adicionar primeiro filamento
   Step 4: Criar primeiro orçamento
   ↓
8. Dashboard completo com dados
```

### 2. Criar Orçamento (Fluxo principal)

```
1. Dashboard → Botão "Novo Orçamento"
   ↓
2. Página: /budgets/new
   
   Step 1: Informações Básicas
   - Nome do projeto
   - Cliente (select ou criar novo)
   - Descrição
   
   Step 2: Tempo de Impressão
   - Horas e minutos
   - Presets (máquina, energia, custo)
   
   Step 3: Filamentos
   - Adicionar filamentos
   - Quantidade em gramas
   - Preview de cores
   
   Step 4: Custos Adicionais
   - Incluir energia? ☑
   - Incluir mão de obra? ☑
   - Incluir desperdício AMS? ☑
   
   Step 5: Informações Extras
   - Prazo de entrega (dias)
   - Condições de pagamento
   - Observações
   
   Step 6: Revisão
   - Card com resumo completo
   - Cálculo automático do total
   - Botões:
     - "Salvar como rascunho"
     - "Salvar e gerar PDF"
   ↓
3. Orçamento criado
   - Toast: "Orçamento criado com sucesso!"
   - Redirect para /budgets/:id
   ↓
4. Página de detalhes do orçamento
   - Visualização completa
   - Ações:
     - Gerar/Baixar PDF
     - Enviar para cliente
     - Mudar status
     - Editar
     - Deletar
```

### 3. Gerenciamento de Assinatura

```
Owner only:

1. Settings → Subscription
   ↓
2. Página mostra:
   - Status atual (Trial / Active / Overdue)
   - Dias restantes do trial
   - Plano atual
   - Histórico de pagamentos
   ↓
3. Opções:
   - Ver planos disponíveis
   - Atualizar método de pagamento (Asaas)
   - Histórico de faturas
   - Cancelar assinatura
```

---

## 🏠 Landing Page

### Estrutura da Landing Page

#### 1. Header (Sticky)
```
Logo | Features | Pricing | Docs | Login | [Começar Grátis]
```

#### 2. Hero Section
```
Background: Branco limpo com sutil overlay cinza neutro (#f7f7f7)

Título: "Gerencie seus orçamentos de impressão 3D de forma profissional"

Subtítulo: "Plataforma completa para criar orçamentos detalhados,
gerenciar clientes e aumentar suas vendas de impressão 3D"

CTAs:
- [Começar agora - 14 dias grátis] (Primary Button - Coral #ff6b6b)
- [Ver demonstração] (Secondary Button - Outline cinza)

Preview: Screenshot animado da plataforma com bordas sutis
```

#### 3. Features Section
```
"Por que escolher o Spooliq?"

Grid 3x2 de features:

📊 Orçamentos Inteligentes
   Cálculo automático considerando filamento, energia e mão de obra

🎨 Catálogo de Filamentos
   Organize todos seus materiais com sistema avançado de cores

📄 PDFs Profissionais
   Geração automática de orçamentos em PDF com sua marca

👥 Gestão de Clientes
   Centralize informações e histórico de cada cliente

📈 Dashboard Analítico
   Acompanhe receita, orçamentos e performance

🔒 Multi-tenancy Seguro
   Seus dados isolados e protegidos
```

#### 4. How It Works
```
"Como funciona?"

3 passos simples:

1️⃣ Cadastre seus produtos
   → Adicione filamentos, materiais e presets

2️⃣ Crie orçamentos
   → Sistema calcula automaticamente todos os custos

3️⃣ Envie para o cliente
   → PDF profissional pronto para enviar
```

#### 5. Pricing
```
"Planos que crescem com você"

Cards de planos (3 colunas):

┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│   Starter       │  │   Professional  │  │   Enterprise    │
│   R$ 29/mês     │  │   R$ 79/mês     │  │   Customizado   │
│                 │  │   [Mais Popular]│  │                 │
│ • 50 orçamentos │  │ • Ilimitado     │  │ • Tudo do Pro   │
│ • 3 usuários    │  │ • 10 usuários   │  │ • Usuários ∞    │
│ • PDF básico    │  │ • PDF completo  │  │ • API acesso    │
│                 │  │ • Logo no PDF   │  │ • Suporte VIP   │
│                 │  │ • Dashboard     │  │ • Onboarding    │
│ [Começar]       │  │ [Começar]       │  │ [Falar conosco] │
└─────────────────┘  └─────────────────┘  └─────────────────┘

✨ Todos os planos incluem 14 dias de teste grátis
```

#### 6. Testimonials
```
"O que nossos clientes dizem"

Carrossel de depoimentos com foto, nome e empresa
```

#### 7. CTA Final
```
Background: Cinza suave neutro (#f7f7f7) com sutil borda superior

"Pronto para profissionalizar seus orçamentos?"

[Começar agora - 14 dias grátis] (Coral #ff6b6b) [Ver demonstração] (Outline)

✓ Sem cartão de crédito para teste
✓ Cancele quando quiser
✓ Suporte em português
```

#### 8. Footer
```
Logo

Produto          Empresa          Recursos          Legal
├─ Features      ├─ Sobre         ├─ Blog          ├─ Termos
├─ Pricing       ├─ Contato       ├─ Docs          ├─ Privacidade
└─ Changelog     └─ Suporte       └─ API           └─ Cookies

Social: LinkedIn | Instagram | YouTube

© 2024 Spooliq. Todos os direitos reservados.
```

---

## 🧩 Componentes Principais

### 1. Layout Components

#### Sidebar
```typescript
// components/layout/sidebar.tsx
interface SidebarProps {
  user: User;
  organizationId: string;
}

// Items do menu baseados em permissão
const menuItems = [
  { icon: LayoutDashboard, label: 'Dashboard', href: '/dashboard', roles: ['all'] },
  { icon: FileText, label: 'Orçamentos', href: '/budgets', roles: ['all'] },
  { icon: Users, label: 'Clientes', href: '/customers', roles: ['all'] },
  { 
    icon: Package,
    label: 'Catálogo',
    children: [
      { label: 'Filamentos', href: '/catalog/filaments' },
      { label: 'Materiais', href: '/catalog/materials' },
      { label: 'Marcas', href: '/catalog/brands' },
    ],
    roles: ['all']
  },
  { icon: Settings, label: 'Presets', href: '/presets', roles: ['all'] },
  { icon: Cog, label: 'Configurações', href: '/settings/company', roles: ['Owner', 'OrgAdmin'] },
]
```

#### TopBar
```typescript
// components/layout/topbar.tsx
- Logo da empresa (se houver)
- Nome da empresa
- Search bar global
- Notifications dropdown
- User menu dropdown
  - Avatar
  - Nome e email
  - Trocar empresa (se tiver mais de uma)
  - Configurações
  - Sair
```

### 2. Budget Components

#### BudgetCard
```typescript
// components/budgets/budget-card.tsx
interface BudgetCardProps {
  budget: Budget;
  onStatusChange?: (status: BudgetStatus) => void;
  onDelete?: () => void;
}

// Card mostrando:
- Nome do projeto
- Cliente
- Status (Badge colorido)
- Valor total (destaque)
- Data de criação
- Ações rápidas (PDF, Editar, Deletar)
```

#### BudgetStatusBadge
```typescript
// components/budgets/status-badge.tsx
const STATUS_CONFIG = {
  draft: { label: 'Rascunho', color: '#9d9d9d', icon: PencilIcon },         // Cinza neutro
  sent: { label: 'Enviado', color: '#0288d1', icon: SendIcon },             // Azul confiável
  approved: { label: 'Aprovado', color: '#00a699', icon: CheckIcon },       // Verde Airbnb
  rejected: { label: 'Rejeitado', color: '#d93025', icon: XIcon },          // Vermelho suave
  printing: { label: 'Imprimindo', color: '#f4a261', icon: PrinterIcon },   // Laranja suave
  completed: { label: 'Concluído', color: '#5a6268', icon: CheckCheckIcon },// Cinza escuro elegante
}
```

#### FilamentSelector
```typescript
// components/budgets/filament-selector.tsx
- Searchable select de filamentos
- Preview da cor (gradient, duo, rainbow)
- Info de marca e material
- Input de quantidade (gramas)
- Cálculo automático do custo
```

### 3. Customer Components

#### CustomerSelect
```typescript
// components/customers/customer-select.tsx
- Combobox com busca
- Opção "Criar novo cliente" inline
- Avatar com iniciais
- Email e telefone (subtitle)
```

### 4. Dashboard Components

#### StatCard
```typescript
// components/dashboard/stat-card.tsx
interface StatCardProps {
  title: string;
  value: string | number;
  icon: LucideIcon;
  trend?: {
    value: number; // +15%
    isPositive: boolean;
  };
  description?: string;
}
```

#### RevenueChart
```typescript
// components/dashboard/revenue-chart.tsx
- Recharts line/bar chart
- Filtros: 7 dias, 30 dias, 3 meses, 1 ano
- Comparação com período anterior
```

#### RecentBudgets
```typescript
// components/dashboard/recent-budgets.tsx
- Lista dos últimos 5 orçamentos
- Link "Ver todos"
```

### 5. Form Components

#### CurrencyInput
```typescript
// components/form/currency-input.tsx
- Formatação automática (R$ 1.234,56)
- Aceita apenas números
- Integra com React Hook Form
```

#### ColorPicker
```typescript
// components/form/color-picker.tsx
- Tipos: solid, gradient, duo, rainbow
- Preview ao vivo
- Hex input + visual picker
```

### 6. Branding/PDF Customization Components

#### BrandingTemplateGallery
```typescript
// components/branding/template-gallery.tsx
interface BrandingTemplateGalleryProps {
  templates: BrandingTemplate[];
  selectedTemplate?: string;
  onSelectTemplate: (template: BrandingTemplate) => void;
}

// Grid de cards com preview de cada template
// Mostra cores principais e mini preview do PDF
```

#### BrandingColorEditor
```typescript
// components/branding/color-editor.tsx
interface BrandingColorEditorProps {
  colors: CompanyBrandingColors;
  onChange: (colors: CompanyBrandingColors) => void;
}

// Accordion com seções:
// - Header Colors
// - Primary Colors  
// - Secondary Colors
// - Text Colors
// - Accent Colors
// - Background Colors
// Cada cor com HEX input + color picker visual
```

#### PDFPreview
```typescript
// components/branding/pdf-preview.tsx
interface PDFPreviewProps {
  branding: CompanyBrandingEntity;
  sampleBudget?: Budget; // Orçamento de exemplo para preview
}

// Preview ao vivo do PDF com as cores aplicadas
// Usa iframe ou canvas para renderizar preview
// Atualiza em tempo real conforme usuário muda cores
```

---

## 📊 Gerenciamento de Estado

### Zustand Stores

#### Auth Store
```typescript
// stores/auth-store.ts
interface AuthStore {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  refreshToken: () => Promise<void>;
  hasRole: (roles: string[]) => boolean;
}
```

#### Company Store
```typescript
// stores/company-store.ts
interface CompanyStore {
  company: Company | null;
  isLoading: boolean;
  
  fetchCompany: () => Promise<void>;
  updateCompany: (data: Partial<Company>) => Promise<void>;
  uploadLogo: (file: File) => Promise<void>;
}
```

#### Branding Store
```typescript
// stores/branding-store.ts
interface BrandingStore {
  branding: CompanyBrandingEntity | null;
  templates: BrandingTemplate[];
  isLoading: boolean;
  
  fetchBranding: () => Promise<void>;
  fetchTemplates: () => Promise<void>;
  updateBranding: (colors: CompanyBrandingColors) => Promise<void>;
  applyTemplate: (templateName: string) => Promise<void>;
}

interface CompanyBrandingColors {
  template_name?: string;
  header_bg_color: string;
  header_text_color: string;
  primary_color: string;
  primary_text_color: string;
  secondary_color: string;
  secondary_text_color: string;
  title_color: string;
  body_text_color: string;
  accent_color: string;
  border_color: string;
  background_color: string;
  table_header_bg_color: string;
  table_row_alt_bg_color: string;
}

interface CompanyBrandingEntity extends CompanyBrandingColors {
  id: string;
  organization_id: string;
  created_at: string;
  updated_at: string;
}

interface BrandingTemplate {
  name: string;
  display_name: string;
  description: string;
  colors: CompanyBrandingEntity;
}
```

### React Query

```typescript
// hooks/queries/use-budgets.ts
export function useBudgets(filters?: BudgetFilters) {
  return useQuery({
    queryKey: ['budgets', filters],
    queryFn: () => budgetService.list(filters),
  });
}

export function useCreateBudget() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: budgetService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['budgets'] });
      toast.success('Orçamento criado com sucesso!');
    },
  });
}

// hooks/queries/use-branding.ts
export function useBranding() {
  return useQuery({
    queryKey: ['branding'],
    queryFn: () => brandingService.get(),
  });
}

export function useBrandingTemplates() {
  return useQuery({
    queryKey: ['branding-templates'],
    queryFn: () => brandingService.listTemplates(),
    staleTime: Infinity, // Templates não mudam
  });
}

export function useUpdateBranding() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: brandingService.update,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['branding'] });
      toast.success('Cores do PDF atualizadas com sucesso!');
    },
  });
}
```

---

## 🎯 Próximos Passos

### Setup Inicial
1. Criar projeto Next.js com TypeScript
2. Configurar Tailwind + shadcn/ui
3. Configurar variáveis de ambiente
4. Setup Axios + React Query
5. Criar estrutura de pastas

### Implementação por Fase

**Fase 1: Auth & Layout**
- [ ] Landing page
- [ ] Login/Register
- [ ] Layout com sidebar
- [ ] Proteção de rotas
- [ ] Auth context/store

**Fase 2: Dashboard & Company**
- [ ] Dashboard principal
- [ ] Configurações da empresa
- [ ] Upload de logo
- [ ] Stats cards

**Fase 3: Customers**
- [ ] Lista de clientes
- [ ] Criar/editar cliente
- [ ] Customer select component

**Fase 4: Catalog (Filaments, Materials, Brands)**
- [ ] CRUD de filamentos
- [ ] Sistema de cores avançado
- [ ] CRUD de materiais e marcas

**Fase 5: Budgets (Core)**
- [ ] Lista de orçamentos
- [ ] Criar orçamento (wizard)
- [ ] Detalhes do orçamento
- [ ] Geração de PDF
- [ ] Mudança de status

**Fase 6: Presets**
- [ ] Presets de máquinas
- [ ] Presets de energia
- [ ] Presets de custos

**Fase 7: User Management**
- [ ] Lista de usuários
- [ ] Convidar usuário
- [ ] Gerenciar permissões

**Fase 8: PDF Branding (Customização de Cores)**
- [ ] Página de configuração de branding
- [ ] Galeria de templates pré-definidos
- [ ] Editor de cores por elemento
- [ ] Color picker component
- [ ] Preview ao vivo do PDF
- [ ] Aplicar template
- [ ] Salvar cores customizadas

**Fase 9: Admin (Platform)**
- [ ] Gerenciar empresas
- [ ] Ver assinaturas
- [ ] Analytics

---

## 📚 Recursos Adicionais

### Documentação
- **Next.js**: https://nextjs.org/docs
- **shadcn/ui**: https://ui.shadcn.com
- **Tailwind CSS**: https://tailwindcss.com/docs
- **React Query**: https://tanstack.com/query
- **Zustand**: https://zustand-demo.pmnd.rs

### Design Inspiration
- https://dribbble.com/tags/saas-dashboard
- https://www.awwwards.com/websites/saas
- https://www.lapa.ninja/

---

## ✨ Considerações Finais

Este documento serve como especificação completa para construção do frontend. 

**Prioridades:**
1. ✅ Experiência do usuário fluida
2. ✅ Design responsivo (mobile-first)
3. ✅ Performance otimizada
4. ✅ Acessibilidade (WCAG 2.1)
5. ✅ SEO para landing page
6. ✅ Testes E2E (Playwright)

**Observações:**
- Todos os textos devem estar em **Português (BR)**
- Valores monetários sempre em **centavos** na API, formatados para R$ no frontend
- Datas em formato ISO 8601, formatadas com date-fns
- Toast notifications para feedback de ações
- Loading states em todas as requisições
- Error boundaries para capturar erros
- Analytics (opcional: Google Analytics, Mixpanel)
- ⚠️ **ATENÇÃO**: Campo `whatsapp` NÃO tem underscore (não é `whats_app`)

---

## 📋 Changelog da Documentação

### v1.3 - 16/10/2024
- ✅ Atualizado Design System com paleta de cores profissional e neutra (inspirada em Airbnb)
- ✅ Substituída paleta rosa vibrante por coral suave (#ff6b6b) como cor primária
- ✅ Adicionado sistema completo de cinzas neutros para maior profissionalismo
- ✅ Incluídas cores accent (teal) e status colors mais sutis
- ✅ Adicionadas variáveis para backgrounds, text colors e borders
- ✅ Atualizada Landing Page para refletir nova identidade visual neutra e profissional
- ✅ Adicionado Guia de Uso das Cores com princípios de design
- ✅ Incluída configuração completa do Tailwind CSS com exemplos práticos
- ✅ Atualizado BudgetStatusBadge com novos códigos de cores profissionais

### v1.2 - 15/10/2024
- ✅ Corrigido RegisterRequest com estrutura flat e campos de endereço obrigatórios
- ✅ Corrigido `whats_app` → `whatsapp` (sem underscore) em todos os endpoints
- ✅ Adicionados campos de subscription faltantes em Company response
- ✅ Adicionada seção de convenções de nomenclatura da API
- ✅ Adicionados endpoints de PDF Branding/Color Customization

### v1.1 - 15/10/2024
- Documentação inicial com todos os endpoints principais

---

**Documento criado em:** 15/10/2024  
**Última atualização:** 16/10/2024 v1.3  
**Versão da API:** v1  
**Backend:** Go 1.21+  
**Frontend Recomendado:** Next.js 14+ com TypeScript

🚀 **Boa sorte na construção do frontend do Spooliq!**

