# typed: strict
# frozen_string_literal: true

class SpaceRepository < ApplicationRepository
  sig { params(space_record: SpaceRecord, can_create_topic: T.nilable(T::Boolean)).returns(Space) }
  def to_model(space_record:, can_create_topic: nil)
    Space.new(
      database_id: space_record.id,
      identifier: space_record.identifier,
      name: space_record.name,
      plan: Plan.deserialize(space_record.plan),
      joined_at: space_record.joined_at,
      can_create_topic:
    )
  end

  sig do
    params(
      space_records: T.any(
        SpaceRecord::PrivateCollectionProxy,
        SpaceRecord::PrivateAssociationRelation
      )
    ).returns(T::Array[Space])
  end
  def to_models(space_records:)
    space_records.map { to_model(space_record: _1) }
  end
end
