import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Echo Trace API 文档',
  description: '多人在线战术对抗游戏开发者文档',
  lang: 'zh-CN',
  
  base: '/',
  
  themeConfig: {
    nav: [
      { text: '首页', link: '/' },
      { text: 'API 参考', link: '/api/' },
      { text: 'Protobuf', link: '/protobuf/' },
      { text: '游戏逻辑', link: '/game-logic/phases' }
    ],
    
    sidebar: {
      '/api/': [
        {
          text: 'WebSocket API',
          items: [
            { text: 'API 概述', link: '/api/' },
            { text: '房间管理', link: '/api/room-management' },
            { text: '游戏操作', link: '/api/game-actions' },
            { text: '状态快照', link: '/api/state-snapshot' },
            { text: 'MCP Server', link: '/api/mcp-server' }
          ]
        }
      ],
      '/protobuf/': [
        {
          text: 'Protobuf 协议',
          items: [
            { text: '协议定义', link: '/protobuf/' },
            { text: '消息类型', link: '/protobuf/message-types' },
            { text: '迁移指南', link: '/protobuf/migration-guide' }
          ]
        }
      ],
      '/game-logic/': [
        {
          text: '游戏逻辑',
          items: [
            { text: '游戏阶段', link: '/game-logic/phases' },
            { text: '道具系统', link: '/game-logic/items' },
            { text: 'AI Boss', link: '/game-logic/ai-boss' },
            { text: '空间索引', link: '/game-logic/spatial-indexing' }
          ]
        }
      ]
    },
    
    socialLinks: [
      { icon: 'github', link: 'https://github.com/your-repo/echo-trace' }
    ],
    
    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2026 Echo Trace Team'
    },
    
    search: {
      provider: 'local'
    }
  }
})
