import React from "react";

const { Label, Textarea } = (window as any).__VIBECMS_SHARED__.ui;

interface FieldGDPROptionsProps {
  field: any;
  updateField: (updates: Record<string, any>) => void;
}

export default function FieldGDPROptions({
  field,
  updateField,
}: FieldGDPROptionsProps) {
  return (
    <div className="space-y-1">
      <Label className="text-[10px] text-slate-500 uppercase">
        Consent Text
      </Label>
      <Textarea
        value={
          field.consent_text ||
          "I agree to the Privacy Policy and consent to having my data stored."
        }
        onChange={(e: any) =>
          updateField({ consent_text: e.target.value })
        }
        className="min-h-[60px] text-sm"
        rows={2}
        placeholder="I agree to the Privacy Policy..."
      />
      <p className="text-[9px] text-slate-400">
        Shown next to the consent checkbox. Include a link to your privacy
        policy.
      </p>
    </div>
  );
}
