import Taro from '@tarojs/taro';

// Only the native picker may supply a local image; never fetch a user-provided URL.
export async function readChosenAvatar(path: string): Promise<string> {
  if (process.env.TARO_ENV !== 'weapp' || !path || /^https?:\/\/(?!tmp\/)/.test(path)) {
    throw new Error('请在微信中选择头像');
  }
  const fs = Taro.getFileSystemManager();
  const info = await new Promise<{ size: number }>((resolve, reject) =>
    fs.getFileInfo({ filePath: path, success: resolve, fail: reject })
  );
  if (info.size < 1 || info.size > 2 * 1024 * 1024) throw new Error('请选择不超过 2 MB 的头像');
  return new Promise<string>((resolve, reject) =>
    fs.readFile({
      filePath: path,
      encoding: 'base64',
      success: (r) => resolve(String(r.data)),
      fail: () => reject(new Error('头像读取失败，请重新选择'))
    })
  );
}

export async function displayAvatar(data: string, userID: string, slot: number): Promise<string> {
  if (!data) return '';
  if (process.env.TARO_ENV !== 'weapp') return `data:image/jpeg;base64,${data}`;
  if (!/^[0-9a-f-]{36}$/i.test(userID)) throw new Error('头像账号信息异常');
  // Two bounded slots let native Image refresh after replacing an avatar.
  const path = `${Taro.env.USER_DATA_PATH}/tidewise-avatar-${userID}-${slot % 2}.jpg`;
  await new Promise<void>((resolve, reject) =>
    Taro.getFileSystemManager().writeFile({
      filePath: path,
      data,
      encoding: 'base64',
      success: () => resolve(),
      fail: () => reject(new Error('头像缓存失败，请重试'))
    })
  );
  return path;
}
