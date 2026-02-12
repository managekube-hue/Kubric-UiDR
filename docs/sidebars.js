/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */

// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  // Main documentation sidebar
  docsSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Tutorial',
      items: [
        'tutorial-basics/create-a-document',
        'tutorial-basics/create-a-blog-post',
        'tutorial-basics/create-a-page',
        'tutorial-basics/markdown-features',
      ],
    },
    {
      type: 'category',
      label: 'Architecture',
      items: [
        'architecture/architecture',
        'architecture/erd_diagram',
      ],
    },
    {
      type: 'category',
      label: 'ITIL',
      items: [
        'itil/ITIL_4_Compliance_Matrix',
        'itil/GMP_Practice_Map',
        'itil/SMP_Practice_Map',
        'itil/TMP_Practice_Map',
      ],
    },
    {
      type: 'category',
      label: 'Specifications',
      items: [
        'srs/core_requirements',
        'srs/ai_orchestration',
        'srs/agent_specs',
      ],
    },
  ],
};

module.exports = sidebars;
