import { OSPF6Area, OSPF6InterfaceConfig } from '../types'

export interface OSPF6AreaRow extends OSPF6Area { _rowId: number }
export interface OSPF6IfaceRow extends OSPF6InterfaceConfig { _rowId: number; name: string }

export { OSPF6AreasPanel } from './OSPF6TabParts/OSPF6AreasPanel'
export { OSPF6GeneralPanel } from './OSPF6TabParts/OSPF6GeneralPanel'
export { OSPF6InterfacesPanel } from './OSPF6TabParts/OSPF6InterfacesPanel'
export { OSPF6Tab } from './OSPF6TabParts/OSPF6Tab'
