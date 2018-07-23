# rabbitmq-timeline
Parse and format logs from RabbitMQ

# Notes

- Only supports RabbitMQ 3.7.0+ using the <a href="https://github.com/erlang-lager/lager">Lager</a> logging framework.

# Install

- Download required binary for your platform:
    - [https://github.com/stephendotcarter/rabbitmq-timeline/releases/](https://github.com/stephendotcarter/rabbitmq-timeline/releases/)

- Make it executable:
    ```
    chmod +x ~/Downloads/rabbitmq-timeline_*
    ```
- Move to a directory in your `PATH`:
    ```
    sudo mv ~/Downloads/rabbitmq-timeline_* /usr/local/bin/rabbitmq-timeline
    ```
- Execute!
    ```
    rabbitmq-timeline
    ```

# Usage

- Run the command followed by 1 or more RabbitMQ logs and redirect the output to a HTML file:
    ```
    rabbitmq-timeline FILE1 FILE2 FILE3... > FILE
    ```
- Example:
    ```
    rabbitmq-timeline rabbit@node1.log rabbit@node2.log rabbit@node3.log > timeline.html
    ```

# Output

- Checkout an example timeline:
    - [testdata/cluster01/timeline.html](http://htmlpreview.github.io/?https://github.com/stephendotcarter/rabbitmq-timeline/blob/master/testdata/cluster01/timeline.html)
