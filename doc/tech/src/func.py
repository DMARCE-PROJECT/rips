def _subscribe(self, topic: str):
    assert isinstance(topic, str)
    if topic in self.__IGNORED_TOPICS:
        return
    self.get_logger().info(f"subscribing to topic {topic}")
    try:
        params = self._customcontext.params_of(topic)
    except:
        self.get_logger().warning(f"can't subscribe to topic {topic}: no such topic")
        return
    if len(params) != 1:
        self.get_logger().warning(f"Topic {topic} has more than one type")
        return
    msgtype = get_message(params[0])

    def f(binmsg):
        pymsg = deserialize_message(binmsg, msgtype)
        self._update_context()
        self._send_msg_event(topic, message_to_yaml(pymsg), binmsg);

	subscription = self.create_subscription(
        msgtype,
        topic,
        f,
        self.__QUEUE_DEPTH,
        raw = True
    )
