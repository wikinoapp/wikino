# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :export_status_record do
    space_record
    export_record
    kind { ExportStatusKind::Queued.serialize }
    changed_at { Time.current }
  end
end
