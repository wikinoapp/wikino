# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :export_status do
    space
    export
    kind { ExportStatusKind::Queued.serialize }
    changed_at { Time.current }
  end
end
