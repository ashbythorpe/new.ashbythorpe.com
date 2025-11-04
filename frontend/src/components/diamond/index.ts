import {
    Component,
    element,
    noShadowRoot,
    staticElement,
} from "../../custom-elements.ts";

@noShadowRoot
@staticElement
@element("diamond-pattern", "./index.svg")
export default class extends Component {}
