/// <reference types="vite/client" />
// This file resolves the error: Cannot find module '@/views/login/index.vue' or its corresponding type declarations. ts(2307)
// This tells TypeScript that all files ending in .vue are Vue components importable via import statements, which usually resolves module-recognition issues.
declare module '*.vue' {
    import { Component } from 'vue'; const component: Component; export default component;
}

declare const __FRONTEND_VERSION__: string;
declare const __FRONTEND_COMMIT__: string;