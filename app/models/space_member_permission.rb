# typed: strict
# frozen_string_literal: true

class SpaceMemberPermission < T::Enum
  enums do
    CreateTopic = new("create:topic")
    CreatePage = new("create:page")
    CreateDraftPage = new("create:draft_page")
  end
end
