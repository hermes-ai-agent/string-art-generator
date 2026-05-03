from PIL import Image
import numpy as np

# Load the mobile rendered image
img = Image.open('docs/OUTPUT_mobile_400px.png').convert('L')
arr = np.array(img)

print('Image shape:', arr.shape)
print('Mean brightness:', np.mean(arr))
print('Min/Max values:', np.min(arr), np.max(arr))
print('Std deviation:', np.std(arr))

# Check for solid black blobs (large areas of very dark pixels)
very_dark = arr < 50
dark_regions = np.sum(very_dark)
print('Very dark pixels (<50):', dark_regions, f'({100*dark_regions/arr.size:.1f}%)')

# Check for empty areas (large areas of very bright pixels)
very_bright = arr > 200
bright_regions = np.sum(very_bright)
print('Very bright pixels (>200):', bright_regions, f'({100*bright_regions/arr.size:.1f}%)')

# Check tonal balance (left vs right)
left_half = arr[:, :arr.shape[1]//2]
right_half = arr[:, arr.shape[1]//2:]
left_mean = np.mean(left_half)
right_mean = np.mean(right_half)
print('Left half mean:', left_mean)
print('Right half mean:', right_mean)
print('L/R balance diff:', abs(left_mean - right_mean))

# Check top vs bottom
top_half = arr[:arr.shape[0]//2, :]
bottom_half = arr[arr.shape[0]//2:, :]
top_mean = np.mean(top_half)
bottom_mean = np.mean(bottom_half)
print('Top half mean:', top_mean)
print('Bottom half mean:', bottom_mean)
print('T/B balance diff:', abs(top_mean - bottom_mean))