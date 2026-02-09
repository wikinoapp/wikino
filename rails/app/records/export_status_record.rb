# typed: strict
# frozen_string_literal: true

class ExportStatusRecord < ApplicationRecord
  self.table_name = "export_statuses"

  enum :kind, {
    ExportStatusKind::Queued.serialize => 0,
    ExportStatusKind::Started.serialize => 1,
    ExportStatusKind::Succeeded.serialize => 2,
    ExportStatusKind::Failed.serialize => 3
  }, prefix: true

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :export_record, foreign_key: :export_id
end
