export async function onRequestPost(context: any) {
  try {
    const formData = await context.request.formData();
    const image = formData.get('image') as File;
    const lines = formData.get('lines') || '10000';
    const alpha = formData.get('alpha') || '0.05';
    const effect = formData.get('effect') || 'v10';
    const stringWidth = formData.get('stringWidth') || '0.18';

    if (!image) {
      return new Response('No image provided', { status: 400 });
    }

    // Convert image to buffer
    const imageBuffer = await image.arrayBuffer();
    const imageBase64 = btoa(String.fromCharCode(...new Uint8Array(imageBuffer)));

    // Call the string-art binary via Worker
    // Since we can't run binaries directly in Cloudflare Pages,
    // we'll need to use a different approach - either:
    // 1. Call an external API that runs the binary
    // 2. Implement the algorithm in JavaScript/WASM
    // 3. Use Cloudflare Workers with Durable Objects
    
    // For now, return an error message
    return new Response(
      JSON.stringify({ 
        error: 'Binary execution not supported in Cloudflare Pages. Need to implement WASM or external API approach.' 
      }), 
      { 
        status: 501,
        headers: { 'Content-Type': 'application/json' }
      }
    );

  } catch (error: any) {
    return new Response(
      JSON.stringify({ error: error.message }), 
      { 
        status: 500,
        headers: { 'Content-Type': 'application/json' }
      }
    );
  }
}
