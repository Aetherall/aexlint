const View = () => <div>{items.map((item) => (item.visible ? <a /> : <b />))}</div>;
