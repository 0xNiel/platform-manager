// web/src/router/index.ts
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: Array<RouteRecordRaw> = [
  {
    path: '/',
    name: 'dashboard',
    component: () => import('../views/DashboardView.vue'),
    meta: { title: 'Dashboard' },
  },
  {
    path: '/tenants',
    name: 'tenants',
    component: () => import('../views/TenantsView.vue'),
    meta: { title: 'Tenants' },
  },
  {
    path: '/tenants/:id',
    name: 'tenant-detail',
    component: () => import('../views/TenantDetailView.vue'),
    props: true,
    meta: { title: 'Tenant Details' },
  },
  {
    path: '/resources',
    name: 'resources',
    component: () => import('../views/ResourcesView.vue'),
    meta: { title: 'Resources' },
  },
  {
    path: '/iam',
    name: 'iam-drift',
    component: () => import('../views/IAMDriftView.vue'),
    meta: { title: 'IAM Drift' },
  },
]

const router = createRouter({
  // Use base path for MFE - this will be /platform/ when embedded
  history: createWebHistory(process.env.BASE_URL || '/'),
  routes,
})

// Update document title on navigation
router.afterEach((to) => {
  const title = to.meta.title as string | undefined
  document.title = title ? `${title} | Platform Manager` : 'Platform Manager'
})

export default router

