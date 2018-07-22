# rabbitmq-timeline
Parse and format logs from RabbitMQ

# Notes

- Only supports RabbitMQ 3.7.0+ using the <a href="https://github.com/erlang-lager/lager">Lager</a> logging framework.

# Install

- Install and configure Go:
    - https://golang.org/doc/install

- Install `rabbitmq-timeline`:
    ```
    go get github.com/stephendotcarter/rabbitmq-timeline
    go install github.com/stephendotcarter/rabbitmq-timeline
    ```

# Usage

- Run the command followed by 1 or more RabbitMQ logs and redirect the output to a HTML file:
    ```
    rabbitmq-timeline LOG_FILE... > timeline.html
    ```
- Example:
    ```
    rabbitmq-timeline rabbit@node1.log rabbit@node2.log rabbit@node3.log > timeline.html
    ```
