export const addTag = (tagName, attr) => {
    const Component = ({ children }) => {
        return <tagName {...attr}>{children}</tagName>
    };

    Component.toString = () => {
        return `<${tagName} ${attr} />`;
    };

    return Component;
};

export default addTag;