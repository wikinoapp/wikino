# typed: false
# frozen_string_literal: true

RSpec.describe Note, type: :model do
  # describe "#titles_in_body" do
  #   it "returns titles" do
  #     [
  #       ["[[a]]", ["a"]],
  #       ["[[ a ]]", [" a "]],
  #       ["[[Hello]]", ["Hello"]],
  #       ["[[こんにちは✌️]]", ["こんにちは✌️"]],
  #       ["[[a]] [[b]]", %w[a b]],
  #       ["[[Hello]] [[World]]", %w[Hello World]],
  #       ["[[こんにちは]] [[世界🌏]]", %w[こんにちは 世界🌏]],
  #       ["[ [a] ]", []],
  #       ["[[a]", []],
  #       # A bit weird, but same behavior as Obsidian, Reflect, Bear and etc.
  #       ["[[[a]]]", ["[a"]],
  #       ["[[[a]]] [[b]]", %w[[a b]]],
  #       ["[[[a]]] [[[b]]]", %w[\[a \[b\]]],
  #       ["[[[ a ]]]", ["[ a "]]
  #     ].each do |(body, expected)|
  #       note = Note.new(body:)
  #       expect(note.titles_in_body).to eq(expected)
  #     end
  #   end
  # end

  describe "#fetch_link_list" do
    context "記事にリンクが含まれているとき" do
      let!(:note_a) { create(:note, modified_at: Time.zone.parse("2024-01-01")) }
      let!(:note_b) { create(:note, modified_at: Time.zone.parse("2024-01-02")) }
      let!(:note_c) { create(:note, linked_note_ids: [note_b.id], modified_at: Time.zone.parse("2024-01-03")) }
      let!(:note_d) { create(:note, linked_note_ids: [note_c.id], modified_at: Time.zone.parse("2024-01-04")) }
      let!(:target_note) { create(:note, linked_note_ids: [note_a.id, note_c.id]) }

      it "リンクの構造体を返すこと" do
        link_list = target_note.fetch_link_list

        expect(link_list.links.size).to eq(2)

        link_a = link_list.links[0]
        expect(link_a.note).to eq(note_c)
        expect(link_a.backlinked_notes.size).to eq(1)
        expect(link_a.backlinked_notes[0]).to eq(note_d)

        link_b = link_list.links[1]
        expect(link_b.note).to eq(note_a)
        expect(link_b.backlinked_notes.size).to eq(0)
      end
    end
  end
end
