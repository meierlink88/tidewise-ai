interface SwitchProps {
  checked: boolean;
  label: string;
  onChange: (checked: boolean) => void;
}

export default function Switch({ checked, label, onChange }: SwitchProps) {
  return (
    <label className="ui-switch-field">
      <span className="ui-field-label">{label}</span>
      <input
        aria-label={label}
        checked={checked}
        className="ui-switch-input"
        onChange={(event) => onChange(event.target.checked)}
        role="switch"
        type="checkbox"
      />
      <span className="ui-switch-track" aria-hidden="true">
        <span className="ui-switch-thumb" />
      </span>
    </label>
  );
}
