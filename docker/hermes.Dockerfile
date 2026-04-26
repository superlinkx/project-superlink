# Replace 'latest' with the specific tag you are using if applicable
FROM nousresearch/hermes-agent:latest

# Switch to root to install system-level browser dependencies
USER root

# Install Playwright/Chrome dependencies
RUN apt update && apt install curl -y

# Clean up apt cache to keep the image size down
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /home/linuxbrew/.linuxbrew && chown -R hermes:hermes /home/linuxbrew

USER hermes
RUN /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"