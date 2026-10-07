import { OSPFArea, OSPFInterfaceConfig } from '../types'

export interface AreaRow extends OSPFArea { _rowId: number }
export interface IfaceRow extends OSPFInterfaceConfig { _rowId: number; name: string }

export const OSPF_AUTH_TYPES = ['none', 'simple', 'md5']

export { OSPFAreasPanel } from './OSPFTabParts/OSPFAreasPanel'
export { OSPFGeneralPanel } from './OSPFTabParts/OSPFGeneralPanel'
export { OSPFInterfacesPanel } from './OSPFTabParts/OSPFInterfacesPanel'
export { OSPFTab } from './OSPFTabParts/OSPFTab'
