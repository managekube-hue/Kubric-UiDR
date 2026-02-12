import React from 'react';
import ComponentCreator from '@docusaurus/ComponentCreator';

export default [
  {
    path: '/compliance',
    component: ComponentCreator('/compliance', 'c3e'),
    exact: true
  },
  {
    path: '/markdown-page',
    component: ComponentCreator('/markdown-page', '2b8'),
    exact: true
  },
  {
    path: '/modules',
    component: ComponentCreator('/modules', '2d4'),
    exact: true
  },
  {
    path: '/',
    component: ComponentCreator('/', '3d1'),
    exact: true
  },
  {
    path: '*',
    component: ComponentCreator('*'),
  },
];
