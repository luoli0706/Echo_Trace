# Item manual data for UI (abbr + usage). Keep in sync with Echo_Trace/Items.md

ITEM_MANUAL = {
    # Offense
    "WPN_SHOCK_T1": {
        "abbr": "SHK",
        "name": "简易电击器",
        "use": "对3m内最近敌人造成伤害并短暂减速。打开背包按数字键使用。",
    },
    "WPN_STONE": {
        "abbr": "STN",
        "name": "投掷石块",
        "use": "低伤害投掷，主要用于制造噪音诱导。",
    },
    "WPN_KNIFE_T2": {
        "abbr": "KNF",
        "name": "战术飞刀",
        "use": "对6m内最近敌人造成高伤害，偏无声。",
    },
    "WPN_STUN_GRENADE": {
        "abbr": "STG",
        "name": "闪光震撼弹",
        "use": "对范围内最近敌人造成伤害（当前为简化实现）。",
    },
    "WPN_TRACK_DART": {
        "abbr": "DRT",
        "name": "追踪毒镖",
        "use": "对6m内最近敌人造成伤害（流血/足迹为后续增强）。",
    },
    "WPN_EMP_MINE": {
        "abbr": "EMP",
        "name": "电磁脉冲雷",
        "use": "对8m内最近敌人造成高伤害（禁用侦察为后续增强）。",
    },

    # Survival
    "SURV_BANDAGE": {
        "abbr": "BDG",
        "name": "急救绷带",
        "use": "回复生命值（当前为即时治疗，后续可加入引导）。",
    },
    "SURV_ENERGY_BAR": {
        "abbr": "BAR",
        "name": "能量棒",
        "use": "少量即时治疗（轻盈状态为后续增强）。",
    },
    "SURV_ADRENALINE": {
        "abbr": "ADR",
        "name": "肾上腺素",
        "use": "短时间提升移动速度（解除控制/虚弱为后续增强）。",
    },
    "SURV_SILENT_PAD": {
        "abbr": "PAD",
        "name": "消音鞋垫",
        "use": "使用后20秒内脚步声大幅降低（更难被听觉侦测）。打开背包按数字键使用。",
    },
    "SURV_JAMMER": {
        "abbr": "JAM",
        "name": "便携式干扰器",
        "use": "使用后12秒内你的脚步声方向会被干扰（对方听觉提示更不准确）。打开背包按数字键使用。",
    },
    "SURV_ARMOR_LIGHT": {
        "abbr": "ARM",
        "name": "凯夫拉内衬",
        "use": "使用后20秒内受到的伤害降低（当前为简化实现：减伤35%）。打开背包按数字键使用。",
    },

    # Recon
    "RECON_AMP_T1": {
        "abbr": "AMP",
        "name": "听音增幅器",
        "use": "临时扩大听觉半径。",
    },
    "RECON_FLASHLIGHT": {
        "abbr": "LGT",
        "name": "定向手电筒",
        "use": "临时提升视距（当前为简化：增加视野半径）。",
    },
    "RECON_HEARTBEAT": {
        "abbr": "HBT",
        "name": "心跳探测仪",
        "use": "临时提升侦察能力（当前为简化：增加视野半径）。",
    },
    "RECON_DRONE_TAG": {
        "abbr": "DRN",
        "name": "无人机信标",
        "use": "标记/高亮待接入（当前为简化：增加视野半径）。",
    },
    "RECON_GLOBAL_SCAN": {
        "abbr": "GSC",
        "name": "全境扫描终端",
        "use": "全图广播待接入（当前为简化：增加视野半径）。",
    },
    "RECON_XRAY": {
        "abbr": "XRY",
        "name": "透视护目镜",
        "use": "透视轮廓待接入（当前为简化：短时增加视野半径）。",
    },

    # Scavenge
    "SCAV_BACKPACK_M": {
        "abbr": "BPK",
        "name": "大容量背包",
        "use": "使用后30秒内背包上限+2，同时最大负重+3。到期后若携带物品数超过上限，无法再拾取直到减少。",
    },
    "SCAV_DETECTOR": {
        "abbr": "DET",
        "name": "金属探测器",
        "use": "高亮附近掉落（当前为简化：给予少量资金）。",
    },
    "SCAV_DECODER": {
        "abbr": "DCD",
        "name": "电机解码卡",
        "use": "与电机交互时使用：立即推进25%进度。",
    },
    "SCAV_MASTER_KEY": {
        "abbr": "KEY",
        "name": "万能钥匙",
        "use": "开启上锁补给箱（当前为简化：给予少量资金）。",
    },
    
    # New Backend Items
    "WPN_AP_AMMO": {"abbr": "AP", "name": "穿甲弹", "use": "装填后下3次射击伤害提升(40)，穿透护甲。"},
    "WPN_BOUNCE_AMMO": {"abbr": "BNC", "name": "跳弹", "use": "装填后下3次射击可弹射多次，伤害25。"},
    "WPN_RAILGUN": {"abbr": "RGN", "name": "磁轨炮", "use": "装填后下1次射击为即时命中高伤激光(75)。"},
    "SURV_REPAIR": {"abbr": "REP", "name": "维修包", "use": "回复30%生命值并修复20点护甲。"},
    "SURV_PHASE_SHIFT": {"abbr": "PHS", "name": "相位转移", "use": "3秒内免疫所有伤害。"},
    "SURV_PURGE": {"abbr": "PUR", "name": "净化系统", "use": "清除负面状态并短时无敌。"},
    "RECON_SCOPE": {"abbr": "SCP", "name": "战术瞄具", "use": "15秒内视野半径增加20%。"},
    "RECON_SENSOR": {"abbr": "SNR", "name": "后视传感器", "use": "10秒内获得后方视野。"},
    "RECON_SCANNER": {"abbr": "SCN", "name": "全域扫描", "use": "向所有人广播最近敌人的位置。"},
    "UTIL_BLINK": {"abbr": "BLK", "name": "闪烁模块", "use": "向朝向瞬间传送2米。"},
    "UTIL_RADAR": {"abbr": "RAD", "name": "便携雷达", "use": "显示周围单位位置（未实装）。"},
    "UTIL_STEALTH": {"abbr": "STH", "name": "光学迷彩", "use": "15秒内隐形（攻击会显形）。"},
    "UTIL_WISH_MACHINE": {"abbr": "WIS", "name": "万能许愿机", "use": "向神明（AI）许愿。"},
}

CATEGORY_ORDER = [
    ("攻击类", ["WPN_SHOCK_T1", "WPN_STONE", "WPN_KNIFE_T2", "WPN_STUN_GRENADE", "WPN_TRACK_DART", "WPN_EMP_MINE", "WPN_AP_AMMO", "WPN_BOUNCE_AMMO", "WPN_RAILGUN"]),
    ("生存类", ["SURV_BANDAGE", "SURV_ENERGY_BAR", "SURV_ADRENALINE", "SURV_SILENT_PAD", "SURV_JAMMER", "SURV_ARMOR_LIGHT", "SURV_REPAIR", "SURV_PHASE_SHIFT", "SURV_PURGE"]),
    ("侦察类", ["RECON_AMP_T1", "RECON_FLASHLIGHT", "RECON_HEARTBEAT", "RECON_DRONE_TAG", "RECON_GLOBAL_SCAN", "RECON_XRAY", "RECON_SCOPE", "RECON_SENSOR", "RECON_SCANNER"]),
    ("搜索类", ["SCAV_BACKPACK_M", "SCAV_DETECTOR", "SCAV_DECODER", "SCAV_MASTER_KEY", "UTIL_BLINK", "UTIL_RADAR", "UTIL_STEALTH", "UTIL_WISH_MACHINE"]),
]

def get_item_abbr(item_id: str) -> str:
    v = ITEM_MANUAL.get(item_id)
    if not v:
        return item_id[:3]
    return v.get("abbr", item_id[:3])


def get_item_name(item_id: str) -> str:
    v = ITEM_MANUAL.get(item_id)
    if not v:
        return item_id
    return v.get("name", item_id)


def get_item_use(item_id: str) -> str:
    v = ITEM_MANUAL.get(item_id)
    if not v:
        return ""
    return v.get("use", "")
