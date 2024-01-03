# Deploy on your local machine

The application can be deployed locally as a compute-local forwarder, similar to the [GoLinks project]. To do this:

1. Build the application

    ```bash
    task bin/all
    ```

2. Move it to a suitable directory

    ```bash
    mv dist/linux+<architecture>/x40.link /usr/local/bin/
    ```

3. Write some URLs to a place that the application can read

    ```bash
    mkdir -p /etc/x40.link
    cat <<'EOF'>> /etc/x40.link/urls.yaml
    ---
    - from: //x40/foo
      to: //k3s/bar
    - from: //x40/bar
      to: //k3s/baz
    EOF
    ```

3. Enable the binary to bind ports lower than 1024 without needing root privileges (Linux Only)

    ```bash
    setcap 'cap_net_bind_service=+ep' /usr/local/bin/x40.link
    ```

4. Create a systemd unit to manage the application

    ```bash
    cat <<'EOF' > /etc/systemd/system/x40.link.service
    [Unit]
    Description="The @link Shortener"
    After=network-online.target

    [Service]
    ExecStart=/usr/local/bin/x40.link redirect serve --with-yaml /etc/x40.link/urls.yaml

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
    systemctl start x40.link && systemctl enable x40.link
    ```

7. Add an entry in the "/etc/hosts" file pointing at localhost, with an appropriate prefix

    ```bash
    # DESTRUCTIVE ACTION. Take due care, or use vim.
    cat <<'EOF' | tee -a /etc/hosts
    
    127.0.0.1 x40
    EOF
    ```

8. Navigate to `http://x40` in your browser. It'll probably warn you about HTTPS, but you can click through that.

[GoLinks project]: https://github.com/GoLinks/golinks