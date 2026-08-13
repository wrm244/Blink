//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework CoreAudio -framework CoreFoundation

#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <unistd.h>

// pm_audio_state 遍历系统音频进程对象，一次查询返回两个状态：
// 返回值的 bit0 = 是否有进程正在运行输入（麦克风）音频流，
// bit1 = 是否有进程正在运行输出（扬声器）音频流。
//
// 使用的 API（macOS 10.15+）：
//   - kAudioHardwarePropertyProcessObjectList —— 所有连接音频系统的客户端进程
//   - kAudioProcessPropertyIsRunningInput / IsRunningOutput —— 该进程是否有
//     活跃的输入/输出 IO。
//
// 注意：旧版 SDK 的 kAudioHardwarePropertyProcessInputList /
// ProcessOutputList 已在新版 macOS 中移除，现在统一通过 ProcessObjectList
// 枚举后逐进程查询。查询的是系统级 HAL 状态，无需麦克风/音频权限，
// 也不会捕获音频内容。
static int pm_audio_state(void) {
    AudioObjectPropertyAddress addr = {
        kAudioHardwarePropertyProcessObjectList,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    UInt32 size = 0;
    OSStatus err = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &addr, 0, NULL, &size);
    if (err != noErr || size == 0) {
        return 0;
    }
    UInt32 count = size / sizeof(AudioObjectID);
    AudioObjectID *ids = (AudioObjectID *)malloc(size);
    if (ids == NULL) {
        return 0;
    }
    err = AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, ids);
    if (err != noErr) {
        free(ids);
        return 0;
    }
    int state = 0;
    for (UInt32 i = 0; i < count; i++) {
        // 跳过本进程自身：Blink 播放的提示音不应被算作"媒体播放"。
        pid_t pid = -1;
        AudioObjectPropertyAddress paddr = {
            kAudioProcessPropertyPID,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        UInt32 psize = sizeof(pid);
        if (AudioObjectGetPropertyData(ids[i], &paddr, 0, NULL, &psize, &pid) == noErr && pid == getpid()) {
            continue;
        }
        AudioObjectPropertyAddress raddr = {
            kAudioProcessPropertyIsRunningInput,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        UInt32 running = 0, rsize = sizeof(running);
        if (AudioObjectGetPropertyData(ids[i], &raddr, 0, NULL, &rsize, &running) == noErr && running) {
            state |= 1;
        }
        raddr.mSelector = kAudioProcessPropertyIsRunningOutput;
        if (AudioObjectGetPropertyData(ids[i], &raddr, 0, NULL, &rsize, &running) == noErr && running) {
            state |= 2;
        }
    }
    free(ids);
    return state;
}
*/
import "C"

// AudioActivity 一次遍历返回两个检测结果：
// meeting 报告当前是否有任何应用正在使用麦克风（输入音频流活跃），
// 用于推断视频会议/通话正在进行。纯收听（未开麦）的会议无法被检测到。
// media 报告当前是否有任何应用正在输出音频，用于推断媒体
// （视频/音乐）正在播放。无声播放的视频无法被检测到。
func AudioActivity() (meeting, media bool) {
	s := C.pm_audio_state()
	return s&1 != 0, s&2 != 0
}
