use anyhow::Result;
use tokio::{net::{TcpListener, TcpStream}, io::{AsyncReadExt}};
use clap::Parser;

#[derive(Parser, Debug, Clone)]
#[command(version, about, long_about = None)]
struct Args {
    /// Name of the person to greet
    #[arg(short, long)]
    port: u16
}

impl ToString for Args {
    // Required method
    fn to_string(&self) -> String {
        return format!("127.0.0.1:{}", self.port);
    }
}

async fn pipe_conn(mut tcp: TcpStream, args: &Args) -> Result<()> {
    println!("connecting to {}", args.to_string());
    let mut to_conn = TcpStream::connect(args.to_string()).await?;
    println!("connection established");
    //let (mut to_read, mut to_write) = to_conn.into_split();
    //let (mut from_read, mut from_write) = tcp.into_split();

    _ = tokio::io::copy_bidirectional(&mut tcp, &mut to_conn).await;

    return Ok(())
}

#[tokio::main]
async fn main() -> Result<()> {
    let args: &'static Args = Box::leak(Box::new(Args::parse()));
    let listener = TcpListener::bind("127.0.0.1:7878").await?;

    while let Ok((tcp, _)) = listener.accept().await {
        println!("i got a connection");
        tokio::spawn(pipe_conn(tcp, args));
    }

    return Ok(())
}
