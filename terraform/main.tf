# ==============================================================================
# Configuración de Infraestructura como Código (IaC) - AWS
# Propósito: Aprovisionamiento de servidor, reglas de red y dependencias base
# ==============================================================================
#Test
terraform {
  # Almacenamiento remoto del archivo de estado (tfstate) 
  backend "s3" {
    bucket = "terraform-estado-p1-bases-998877" # Asegurar correspondencia con el bucket 
    key    = "infraestructura/terraform.tfstate"
    region = "us-east-1"
  }
}

# Configuración del proveedor de nube
provider "aws" {
  region = "us-east-1" # Región principal 
}

# Definición de reglas de red 
resource "aws_security_group" "proyecto_sg" {
  name        = "p1-bases-sg"
  description = "Permitir SSH, API y Keycloak"

  # Reglas de tráfico entrante
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 8000
    to_port     = 8000
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 8082
    to_port     = 8082
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Tráfico de salida 
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# Provisionamiento del nodo de cómputo (Instancia EC2)
resource "aws_instance" "mi_servidor" {
  ami           = "ami-080e1f13689e07408" # Ubuntu Server 22.04 LTS
  instance_type = "t3.micro"            
  
  # Asociación del Security Group
  vpc_security_group_ids = [aws_security_group.proyecto_sg.id]

  # Configuración de almacenamiento persistente (EBS)
  root_block_device {
    volume_size = 30
    volume_type = "gp3"
  }

  # Cloud-Init / User Data: Script de automatización de arranque 
  user_data = <<-EOF
              #!/bin/bash
              # Configuración de memoria Swap (2GB) para prevenir eventos OOM en instancias pequeñas
              fallocate -l 2G /swapfile
              chmod 600 /swapfile
              mkswap /swapfile
              swapon /swapfile
              echo '/swapfile none swap sw 0 0' | tee -a /etc/fstab

              # Instalación de dependencias del sistema operativo
              apt-get update
              apt-get install -y docker.io docker-compose git

              # Configuración de permisos y habilitacin de Docker
              systemctl enable docker
              systemctl start docker
              usermod -aG docker ubuntu
              EOF

  tags = {
    Name = "P1-Bases-Server"
  }
}

# Variables de salida 
output "ip_publica" {
  value       = aws_instance.mi_servidor.public_ip
  description = "Esta IP se puede usar en Postman para conectarte por SSH"
}