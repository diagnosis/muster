import styles from '@/components/Modal.module.css'

interface ModalProps{
    title: string
    onClose: () => void
    children: React.ReactNode
}


export function Modal({onClose, children, title}: ModalProps){
    return (
        <div className={styles.backdrop} onClick={onClose}>
            <div className={styles.panel} role="dialog" aria-modal={true} aria-labelledby={"modal-title"} onClick={e => e.stopPropagation()}>
                <div className={styles.modalHeader}>
                    <h3 id="modal-title" className={styles.subheading}>{title}</h3>
                    <button aria-label={'Close'} className={styles.closeBtn} onClick={onClose}>✕</button>
                </div>
                {children}
            </div>
        </div>
    )
}