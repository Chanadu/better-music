import type { DatabaseData, RecordSyncStatus } from '$lib/scripts/types';

// Scoped to specimens; never changes the application's library or stored preferences.
export const previewContext = Symbol('visual-preview');
export type PreviewContext = {
	inlineDialogs: true;
	library?: DatabaseData;
	recordStatus?: RecordSyncStatus;
	online?: boolean;
	nativeDropdowns?: boolean;
	newlyAdded?: boolean;
	accountAction?: 'email' | 'password' | 'delete';
};
