/**
 * Below are the colors that are used in the app. The colors are defined in the light and dark mode.
 * There are many other ways to style your app. For example, [Nativewind](https://www.nativewind.dev/), [Tamagui](https://tamagui.dev/), [unistyles](https://reactnativeunistyles.vercel.app), etc.
 */

import { Platform } from 'react-native';

const tintColorLight = '#0a7ea4';
const tintColorDark = '#fff';

export const Colors = {
  light: {
    text: '#11181C',
    background: '#fff',
    tint: tintColorLight,
    icon: '#687076',
    tabIconDefault: '#687076',
    tabIconSelected: tintColorLight,
    // Chat
    bubbleMe: '#dcf8c6',
    bubbleOther: '#ffffff',
    bubbleText: '#111111',
    bubbleTime: '#999999',
    bubbleTimeMeColor: '#7aab6e',
    inputBar: '#f0f0f0',
    inputBarBorder: '#dddddd',
    inputBg: '#ffffff',
    inputText: '#111111',
    inputBorder: '#e0e0e0',
    inputPlaceholder: '#999999',
    // Agora
    noteAuthor: '#555555',
    noteTime: '#888888',
    noteFooterBorder: 'rgba(0,0,0,0.08)',
    // Login
    pageBg: '#f0f4f8',
    cardBg: '#ffffff',
    cardBorder: 'rgba(0,0,0,0.12)',
    textPrimary: 'rgba(0,0,0,0.87)',
    textSecondary: 'rgba(0,0,0,0.6)',
    inputBorderLogin: 'rgba(0,0,0,0.23)',
    dividerLine: 'rgba(0,0,0,0.12)',
    // Eval status
    evalTodo: '#6366F1',
    evalDone: '#16A34A',
    evalInProgress: '#F97316',
    evalTodoBg: '#EEF2FF',
    evalTodoBorder: '#A5B4FC',
    evalDoneBg: '#F0FDF4',
    evalDoneBorder: '#86EFAC',
    // Évaluation envoyée, mais dont les verbatims attendent la relecture
    evalModerationBg: '#FFF7ED',
    evalModerationBorder: '#FDBA74',
    // Évaluation envoyée, dont au moins un verbatim a été refusé
    evalRejectedBg: '#FEF2F2',
    evalRejectedBorder: '#FCA5A5',
    // Section accents
    sectionPeda: '#6366F1',
    sectionContent: '#0EA5E9',
    sectionCharge: '#F97316',
    sectionSupports: '#8B5CF6',
    sectionAmbiance: '#22C55E',
    // Zen background
    zenBase: '#eef5f0',
    zenBlob1: '#5BA67A',
    zenBlob2: '#4A9CC4',
    zenBlob3: '#C4956A',
    zenBlob4: '#8B7ED4',
    zenBlob5: '#5BA67A',
  },
  dark: {
    text: '#ECEDEE',
    background: '#151718',
    tint: tintColorDark,
    icon: '#9BA1A6',
    tabIconDefault: '#9BA1A6',
    tabIconSelected: tintColorDark,
    // Chat
    bubbleMe: '#1e4620',
    bubbleOther: '#2a2a2a',
    bubbleText: '#eeeeee',
    bubbleTime: '#777777',
    bubbleTimeMeColor: '#5a8a5e',
    inputBar: '#1a1a1a',
    inputBarBorder: '#333333',
    inputBg: '#2a2a2a',
    inputText: '#eeeeee',
    inputBorder: '#444444',
    inputPlaceholder: '#666666',
    // Agora
    noteAuthor: '#aaaaaa',
    noteTime: '#888888',
    noteFooterBorder: 'rgba(255,255,255,0.1)',
    // Login
    pageBg: '#0d1117',
    cardBg: '#1c1c1e',
    cardBorder: 'rgba(255,255,255,0.1)',
    textPrimary: 'rgba(255,255,255,0.87)',
    textSecondary: 'rgba(255,255,255,0.5)',
    inputBorderLogin: 'rgba(255,255,255,0.23)',
    dividerLine: 'rgba(255,255,255,0.12)',
    // Eval status
    evalTodo: '#818CF8',
    evalDone: '#4ADE80',
    evalInProgress: '#FB923C',
    evalTodoBg: '#1e1b4b',
    evalTodoBorder: '#6366F1',
    evalDoneBg: '#052e16',
    evalDoneBorder: '#16A34A',
    // Évaluation envoyée, mais dont les verbatims attendent la relecture
    evalModerationBg: '#431407',
    evalModerationBorder: '#F97316',
    // Évaluation envoyée, dont au moins un verbatim a été refusé
    evalRejectedBg: '#450a0a',
    evalRejectedBorder: '#EF4444',
    // Section accents
    sectionPeda: '#6366F1',
    sectionContent: '#0EA5E9',
    sectionCharge: '#F97316',
    sectionSupports: '#8B5CF6',
    sectionAmbiance: '#22C55E',
    // Zen background
    zenBase: '#0f1a12',
    zenBlob1: '#2a4a2e',
    zenBlob2: '#1a3a4a',
    zenBlob3: '#3a2a1a',
    zenBlob4: '#2a2240',
    zenBlob5: '#2a4a2e',
  },
};

export const Fonts = Platform.select({
  ios: {
    /** iOS `UIFontDescriptorSystemDesignDefault` */
    sans: 'system-ui',
    /** iOS `UIFontDescriptorSystemDesignSerif` */
    serif: 'ui-serif',
    /** iOS `UIFontDescriptorSystemDesignRounded` */
    rounded: 'ui-rounded',
    /** iOS `UIFontDescriptorSystemDesignMonospaced` */
    mono: 'ui-monospace',
  },
  default: {
    sans: 'normal',
    serif: 'serif',
    rounded: 'normal',
    mono: 'monospace',
  },
  web: {
    sans: "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif",
    serif: "Georgia, 'Times New Roman', serif",
    rounded: "'SF Pro Rounded', 'Hiragino Maru Gothic ProN', Meiryo, 'MS PGothic', sans-serif",
    mono: "SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace",
  },
});
