# typed: strict
# frozen_string_literal: true

module Spaces
  module Settings
    module Exports
      module Logs
        class IndexView < ApplicationView
          sig do
            params(
              current_user_entity: UserEntity,
              space_entity: SpaceEntity,
              export_status_entity: ExportStatusEntity,
              export_log_entities: T::Array[ExportLogEntity]
            ).void
          end
          def initialize(
            current_user_entity:,
            space_entity:,
            export_status_entity:,
            export_log_entities:
          )
            @current_user_entity = current_user_entity
            @space_entity = space_entity
            @export_status_entity = export_status_entity
            @export_log_entities = export_log_entities
          end

          private def needs_status_watch?
            export_status_entity.kind == ExportStatusKind::Queued ||
              export_status_entity.kind == ExportStatusKind::Started
          end

          sig { returns(UserEntity) }
          attr_reader :current_user_entity
          private :current_user_entity

          sig { returns(SpaceEntity) }
          attr_reader :space_entity
          private :space_entity

          sig { returns(ExportStatusEntity) }
          attr_reader :export_status_entity
          private :export_status_entity

          sig { returns(T::Array[ExportLogEntity]) }
          attr_reader :export_log_entities
          private :export_log_entities
        end
      end
    end
  end
end
