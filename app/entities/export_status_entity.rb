# typed: strict
# frozen_string_literal: true

class ExportStatusEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, T::Wikino::DatabaseId
  const :kind, ExportStatusKind
  const :changed_at, ActiveSupport::TimeWithZone
  const :space_entity, SpaceEntity
  const :export_entity, ExportEntity

  sig { returns(T::Boolean) }
  def processing?
    kind == ExportStatusKind::Queued || kind == ExportStatusKind::Started
  end

  sig { returns(T::Boolean) }
  def finished?
    kind == ExportStatusKind::Succeeded || kind == ExportStatusKind::Failed
  end

  sig { returns(T::Boolean) }
  def succeeded?
    kind == ExportStatusKind::Succeeded
  end

  sig { returns(T::Boolean) }
  def failed?
    kind == ExportStatusKind::Failed
  end
end
