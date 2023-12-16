# Deployment

There are various ways in which this application can be used. They'll be documented here as they're supported.

## Local

The application can be deployed locally as a compute-local forwarder, similar to the [GoLinks project]. To do this:

1. Build the application

    ```bash
    go build
    ```

2. Move it to a suitable directory

    ```bash
    mv s3k.link /usr/local/bin/
    ```

3. Write some URLs to a place that the application can read

    ```bash
    mkdir -p /etc/s3k.link
    cat <<'EOF'>> /etc/s3k.link/urls.yaml
    ---
    - from: //s3k/foo
      to: //k3s/bar
    - from: //s3k/bar
      to: //k3s/baz
    EOF
    ```

3. Enable the binary to bind ports lower than 1024 without needing root privileges (Linux Only)

    ```bash
    setcap 'cap_net_bind_service=+ep' /usr/local/bin/s3k.link
    ```

4. Create a systemd unit to manage the application

    ```bash
    cat <<'EOF' > /etc/systemd/system/s3k.link.service
    [Unit]
    Description="The Skink Link Shortener"
    After=network-online.target

    [Service]
    ExecStart=/usr/local/bin/s3k.link redirect serve --with-yaml /etc/s3k.link/urls.yaml

    [Install]
    WantedBy=multi-user.target
    EOF
    ```

5. Reload systemd

    ```bash
    systemctl daemon-reload
    ```

6. Start, and enable (at boot) the service

    ```bash
    systemctl start s3k.link && systemctl enable s3k.link
    ```

7. Add an entry in the "/etc/hosts" file pointing at localhost, with an appropriate prefix

    ```bash
    # DESTRUCTIVE ACTION. Take due care, or use vim.
    cat <<'EOF' | tee -a /etc/hosts
    
    127.0.0.1 s3k
    EOF
    ```

8. Navigate to http://s3k in your browser. It'll probably warn you about HTTPS, but you can click through that.

[GoLinks project]: https://github.com/GoLinks/golinks