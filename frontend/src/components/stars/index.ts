import {
    Component,
    element,
    noShadowRoot,
    staticElement,
} from "../../custom-elements.ts";

@noShadowRoot
@staticElement
@element("star-pattern", "./index.svg")
export default class extends Component {}
