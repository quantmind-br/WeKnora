// When a folder is dragged, Chrome/Edge fill dataTransfer.files with "fake"
// directory entries (size 0, no extension) instead of the real files inside the
// folder. Only webkitGetAsEntry() can walk a dropped directory recursively, so it
// must be tried first, falling back to dataTransfer.files only when the entry API
// is unavailable. Firefox leaves dataTransfer.files empty when a folder is dragged,
// so it also needs the entry traversal path.

const isHiddenSegment = (segment: string): boolean => segment.startsWith('.')

export const setRelativePath = (file: File, relativePath: string): void => {
  try {
    Object.defineProperty(file, 'webkitRelativePath', {
      value: relativePath,
      writable: false,
      enumerable: true,
      configurable: true,
    })
  } catch {
    // Older Safari may refuse defineProperty on File; those files
    // carry no relative path and are uploaded flat.
  }
}

const readAllDirEntries = (reader: { readEntries: Function }): Promise<any[]> => {
  return new Promise((resolve) => {
    const collected: any[] = []
    const readBatch = () => {
      reader.readEntries((entries: any[]) => {
        if (!entries || entries.length === 0) {
          resolve(collected)
        } else {
          collected.push(...entries)
          readBatch()
        }
      }, () => resolve(collected))
    }
    readBatch()
  })
}

export const traverseEntry = (entry: any, path: string): Promise<File[]> => {
  return new Promise((resolve) => {
    try {
      if (!entry) {
        resolve([])
        return
      }
      if (entry.isFile) {
        if (typeof entry.file !== 'function') {
          resolve([])
          return
        }
        entry.file((file: File) => {
          // Only set webkitRelativePath on files inside a directory, matching
          // <input webkitdirectory> behavior. Files dropped at the top level
          // keep an empty value and upload as plain files.
          if (path) {
            const relativePath = `${path}/${file.name}`
            if (relativePath.split('/').some(isHiddenSegment)) {
              resolve([])
              return
            }
            setRelativePath(file, relativePath)
          }
          resolve([file])
        }, () => resolve([]))
      } else if (entry.isDirectory) {
        const dirPath = path ? `${path}/${entry.name}` : entry.name
        // Skip hidden directories (.git, .DS_Store, etc.)
        if (dirPath.split('/').some(isHiddenSegment)) {
          resolve([])
          return
        }
        if (typeof entry.createReader !== 'function') {
          resolve([])
          return
        }
        readAllDirEntries(entry.createReader())
          .then(children => Promise.all(
            children.map(c => traverseEntry(c, dirPath).catch(() => [] as File[])),
          ))
          .then(results => resolve(results.flat()))
          .catch(() => resolve([]))
      } else {
        resolve([])
      }
    } catch {
      resolve([])
    }
  })
}

export const collectDroppedFiles = async (event: DragEvent): Promise<File[]> => {
  const dataTransfer = event.dataTransfer
  const items = dataTransfer?.items ? Array.from(dataTransfer.items) : []
  // DataTransfer is only guaranteed during the synchronous drop phase, so copy the fallback list first.
  const fallbackFiles = dataTransfer?.files ? Array.from(dataTransfer.files) : []

  if (items.length === 0) {
    return fallbackFiles
  }

  const fileItems = items.filter(item => item.kind === 'file')
  if (fileItems.length === 0) {
    return fallbackFiles
  }

  const pairs = fileItems.map(item => {
    try {
      return { item, entry: (item as any).webkitGetAsEntry?.() ?? null }
    } catch {
      return { item, entry: null }
    }
  })
  const usable = pairs.filter(p => p.entry != null)
  if (usable.length === 0) {
    // The browser does not support webkitGetAsEntry; fall back to the snapshotted FileList.
    return fallbackFiles
  }

  const results = await Promise.all(usable.map(async ({ item, entry }) => {
    try {
      if (entry.isDirectory) {
        return await traverseEntry(entry, '')
      }
      // Top-level files use getAsFile: synchronous, and keeps webkitRelativePath empty.
      const file = item.getAsFile()
      if (file) return [file]
      return await traverseEntry(entry, '')
    } catch {
      if (entry?.isDirectory) return []
      const file = item.getAsFile()
      return file ? [file] : []
    }
  }))

  // Once a FileSystemEntry was obtained, trust the traversal result (including an empty array).
  // Falling back to dataTransfer.files for an empty folder, or one holding only hidden files,
  // would make Chrome/Edge hand over the size-0 phantom directory entries again.
  return results.flat()
}
