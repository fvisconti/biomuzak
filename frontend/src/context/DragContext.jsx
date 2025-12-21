import React, { createContext, useContext, useRef } from 'react';

const DragContext = createContext();

export const useDrag = () => useContext(DragContext);

export const DragProvider = ({ children }) => {
    const onDragEndHandler = useRef(null);

    const setOnDragEnd = (handler) => {
        onDragEndHandler.current = handler;
    };

    return (
        <DragContext.Provider value={{ onDragEndHandler, setOnDragEnd }}>
            {children}
        </DragContext.Provider>
    );
};
