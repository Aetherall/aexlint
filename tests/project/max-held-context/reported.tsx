const View = () => <div>{items.map((item) => (item.visible ? (item.open ? (item.big ? <a /> : <b />) : <c />) : <d />))}</div>;
