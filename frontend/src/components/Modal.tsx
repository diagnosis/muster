import styles from '@/components/Modal.module.css'

interface ModalProps{
    onClose: () => void
    children: React.ReactNode
}


export function Modal({onClose, children}: ModalProps){
    return (
        <div className={styles.backdrop} onClick={onClose}>
            <div className={styles.panel} role="dialog" aria-modal={true} onClick={e => e.stopPropagation()}>
                {children}
            </div>
        </div>
    )
}