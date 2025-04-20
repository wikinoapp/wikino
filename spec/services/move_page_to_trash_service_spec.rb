# typed: false
# frozen_string_literal: true

RSpec.describe MovePageToTrashService, type: :use_case do
  describe "#call" do
    it "ページをゴミ箱に移動できること" do
      page = create(:page)
      updated_at = page.updated_at

      expect(page.trashed_at).to be_nil

      result = MovePageToTrashService.new.call(page:)

      expect(result.page).to eq(page)
      expect(result.page.trashed_at).to be_present
      expect(result.page.updated_at).to be > updated_at
    end
  end
end
